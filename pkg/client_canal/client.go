package client_canal

import (
	"fmt"
	"github.com/withlin/canal-go/client"
	pbe "github.com/withlin/canal-go/protocol/entry"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
	"log"
	"orca_trade_data/pkg"
	"os"
	"strings"
	"time"
)

type SimpleClint struct {
	Addr                 string
	Port                 int
	UserName             string
	Password             string
	Destination          string
	SoTimeOut            int32
	IdleTimeOut          int32
	SimpleCanalConnector *client.SimpleCanalConnector
	SlaveDb              *gorm.DB
}

func NewClient(cfg pkg.Config) *SimpleClint {
	mysqlMaster := cfg.MysqlMaster
	simpleClient := SimpleClint{
		Addr:        mysqlMaster.Host,
		Port:        mysqlMaster.Port,
		UserName:    mysqlMaster.MasterUser,
		Password:    mysqlMaster.MasterPass,
		Destination: mysqlMaster.Destination,
		SoTimeOut:   int32(mysqlMaster.SoTimeOut),
		IdleTimeOut: int32(mysqlMaster.IdleTimeOut),
	}
	slave, err := pkg.LoadMysql(cfg)
	if err != nil {
		log.Fatalf("slave conect error: %v", err)
	}
	simpleClient.SlaveDb = slave
	simpleClient.SimpleCanalConnector = client.NewSimpleCanalConnector(simpleClient.Addr, simpleClient.Port, simpleClient.UserName, simpleClient.Password, simpleClient.Destination, simpleClient.SoTimeOut, simpleClient.IdleTimeOut)
	return &simpleClient
}
func (s *SimpleClint) HandleData() error {
	if err := s.Connect(); err != nil {
		return err
	}
	if err := s.Subscribe(pkg.SubscribeDb); err != nil {
		return err
	}
	return nil
}
func (s *SimpleClint) Connect() error {
	if err := s.SimpleCanalConnector.Connect(); err != nil {
		log.Printf("connect canal err: %v", err)
		return err
	}
	return nil
}

// https://github.com/alibaba/canal/wiki/AdminGuide
// mysql 数据解析关注的表，Perl正则表达式.
//
// 多个正则之间以逗号(,)分隔，转义符需要双斜杠(\\)
//
// 常见例子：
//
//  1. 所有表：.*   or  .*\\..*
//  2. canal schema下所有表： canal\\..*
//  3. canal下的以canal打头的表：canal\\.canal.*
//  4. canal schema下的一张表：canal\\.test1
//  5. 多个规则组合使用：canal\\..*,mysql.test1,mysql.test2 (逗号分隔)

func (s *SimpleClint) Subscribe(filter string) error {
	if err := s.SimpleCanalConnector.Subscribe(filter); err != nil {
		log.Printf("connect canal err: %v", err)
		return err
	}
	for {

		message, err := s.SimpleCanalConnector.Get(100, nil, nil)
		if err != nil {
			log.Printf("canal get data err: %v", err)
			return err
		}
		batchId := message.Id
		if batchId == -1 || len(message.Entries) <= 0 {
			time.Sleep(300 * time.Millisecond)
			//fmt.Println("===没有数据了===")
			continue
		}
		// todo
		s.printEntry(message.Entries)

	}
}
func (s *SimpleClint) printEntry(entrys []pbe.Entry) {

	for _, entry := range entrys {
		if entry.GetEntryType() == pbe.EntryType_TRANSACTIONBEGIN || entry.GetEntryType() == pbe.EntryType_TRANSACTIONEND {
			continue
		}
		rowChange := new(pbe.RowChange)

		err := proto.Unmarshal(entry.GetStoreValue(), rowChange)
		checkError(err)
		if rowChange != nil {
			eventType := rowChange.GetEventType()
			header := entry.GetHeader()
			//fmt.Println(fmt.Sprintf("================> binlog[%s : %d],name[%s,%s], eventType: %s", header.GetLogfileName(), header.GetLogfileOffset(), header.GetSchemaName(), header.GetTableName(), header.GetEventType()))
			tableName := header.GetTableName()
			for _, rowData := range rowChange.GetRowDatas() {
				//fmt.Println("data.....................")
				if eventType == pbe.EventType_DELETE {
					//printColumn(rowData.GetBeforeColumns())
					s.DeleteData(pbe.EventType_DELETE, tableName, rowData.GetBeforeColumns())
				} else if eventType == pbe.EventType_INSERT {
					//printColumn(rowData.GetAfterColumns())
					s.InsertData(pbe.EventType_INSERT, tableName, rowData.GetAfterColumns())
				} else {
					//fmt.Println("-------> before")
					//printColumn(rowData.GetBeforeColumns())
					//fmt.Println("-------> after")
					//printColumn(rowData.GetAfterColumns())
					s.UpdateData(pbe.EventType_UPDATE, tableName, rowData.GetAfterColumns())
				}
			}
		}
	}
}

func printColumn(columns []*pbe.Column) {
	for _, col := range columns {
		fmt.Println(fmt.Sprintf("%s : %s mysql_type:%s update= %t", col.GetName(), col.GetValue(), col.GetMysqlType(), col.GetUpdated()))
	}
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}

func (s *SimpleClint) InsertData(enventType pbe.EventType, tableName string, columns []*pbe.Column) {
	fieldSli, valueSli := make([]string, 0), make([]string, 0)
	for _, col := range columns {
		if col.GetValue() != "" {
			fieldSli = append(fieldSli, col.GetName())
			valueSli = append(valueSli, fmt.Sprintf("'%s'", col.GetValue()))
		}
	}
	if len(fieldSli) > 0 && len(valueSli) > 0 {
		fieldStr := strings.Join(fieldSli, ",")
		valueStr := strings.Join(valueSli, ",")
		sql := fmt.Sprintf("insert into %s (%s) values (%s)", tableName, fieldStr, valueStr)
		fmt.Println("insert sql:", sql)
		err := s.SlaveDb.Exec(sql).Error
		if err != nil {
			log.Printf("insert data err: %v", err)
			return
		}
	}
}
func (s *SimpleClint) DeleteData(enventType pbe.EventType, tableName string, columns []*pbe.Column) {
	fmt.Println("delete:")
	for _, col := range columns {
		if col.GetIsKey() {
			sql := fmt.Sprintf("delete from %s where %s = %s", tableName, col.GetName(), col.GetValue())
			fmt.Println(sql)
			err := s.SlaveDb.Exec(sql).Error
			if err != nil {
				log.Printf("delete data err: %v", err)
				return
			}
		}
	}
}
func (s *SimpleClint) UpdateData(enventType pbe.EventType, tableName string, columns []*pbe.Column) {
	mp := make(map[string]string)
	priMp := make(map[string]string)
	for _, col := range columns {
		if col.GetIsKey() {
			priMp[col.GetName()] = col.GetValue()
		}
		if col.GetUpdated() {
			mp[col.GetName()] = col.GetValue()
		}
	}
	if len(mp) > 0 {
		ups := make([]string, 0)
		for key, val := range mp {
			ups = append(ups, fmt.Sprintf("%s='%s'", key, val))
		}
		upStr := strings.Join(ups, ",")
		str := ""
		for k, v := range priMp {
			str = fmt.Sprintf("%s = '%s'", k, v)
		}
		sql := fmt.Sprintf("update %s set %s where %s", tableName, upStr, str)
		fmt.Println("update sql:", sql)
		err := s.SlaveDb.Exec(sql).Error
		if err != nil {
			log.Printf("update data err: %v", err)
			return
		}

	}
}
