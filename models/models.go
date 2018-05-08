// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2018 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"

	// Needed for the MySQL driver
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	"gopkg.in/yaml.v2"

	// Needed for the Postgresql driver
	_ "github.com/lib/pq"

	// Needed for the MSSSQL driver
	_ "github.com/denisenkom/go-mssqldb"
)

// Engine represents a xorm engine or session.
type Engine interface {
	Table(tableNameOrBean interface{}) *xorm.Session
	Count(...interface{}) (int64, error)
	Decr(column string, arg ...interface{}) *xorm.Session
	Delete(interface{}) (int64, error)
	Exec(...interface{}) (sql.Result, error)
	Find(interface{}, ...interface{}) error
	Get(interface{}) (bool, error)
	ID(interface{}) *xorm.Session
	In(string, ...interface{}) *xorm.Session
	Incr(column string, arg ...interface{}) *xorm.Session
	Insert(...interface{}) (int64, error)
	InsertOne(interface{}) (int64, error)
	Iterate(interface{}, xorm.IterFunc) error
	Join(joinOperator string, tablename interface{}, condition string, args ...interface{}) *xorm.Session
	SQL(interface{}, ...interface{}) *xorm.Session
	Where(interface{}, ...interface{}) *xorm.Session
}

var (
	x                  *xorm.Engine
	supportedDatabases = []string{"mysql", "postgres", "mssql"}
	tables             []interface{}

	// HasEngine specifies if we have a xorm.Engine
	HasEngine bool

	// DbCfg holds the database settings
	DbCfg struct {
		Type, Host, Name, User, Passwd, Path, SSLMode string
		Timeout                                       int
	}

	// EnableSQLite3 use SQLite3
	EnableSQLite3 bool

	// EnableTiDB enable TiDB
	EnableTiDB bool
)

func init() {
	tables = append(tables,
		new(User),
		new(PublicKey),
		new(AccessToken),
		new(Repository),
		new(DeployKey),
		new(Collaboration),
		new(Access),
		new(Upload),
		new(Watch),
		new(Star),
		new(Follow),
		new(Action),
		new(Issue),
		new(PullRequest),
		new(Comment),
		new(Attachment),
		new(Label),
		new(IssueLabel),
		new(Milestone),
		new(Mirror),
		new(Release),
		new(LoginSource),
		new(Webhook),
		new(HookTask),
		new(Team),
		new(OrgUser),
		new(TeamUser),
		new(TeamRepo),
		new(Notice),
		new(EmailAddress),
		new(Notification),
		new(IssueUser),
		new(LFSMetaObject),
		new(TwoFactor),
		new(GPGKey),
		new(RepoUnit),
		new(RepoRedirect),
		new(ExternalLoginUser),
		new(ProtectedBranch),
		new(UserOpenID),
		new(IssueWatch),
		new(CommitStatus),
		new(Stopwatch),
		new(TrackedTime),
		new(DeletedBranch),
		new(RepoIndexerStatus),
		new(IssueDependency),
		new(LFSLock),
		new(Reaction),
		new(IssueAssignees),
		new(U2FRegistration),
		new(TeamUnit),
		new(Review),
		new(OAuth2Application),
		new(OAuth2AuthorizationCode),
		new(OAuth2Grant),
	)

	gonicNames := []string{"SSL", "UID"}
	for _, name := range gonicNames {
		core.LintGonicMapper[name] = true
	}
}

// LoadConfigs loads the database settings
func LoadConfigs() {
	sec := setting.Cfg.Section("database")
	DbCfg.Type = sec.Key("DB_TYPE").String()
	switch DbCfg.Type {
	case "sqlite3":
		setting.UseSQLite3 = true
	case "mysql":
		setting.UseMySQL = true
	case "postgres":
		setting.UsePostgreSQL = true
	case "tidb":
		setting.UseTiDB = true
	case "mssql":
		setting.UseMSSQL = true
	}
	DbCfg.Host = sec.Key("HOST").String()
	DbCfg.Name = sec.Key("NAME").String()
	DbCfg.User = sec.Key("USER").String()
	if len(DbCfg.Passwd) == 0 {
		DbCfg.Passwd = sec.Key("PASSWD").String()
	}
	DbCfg.SSLMode = sec.Key("SSL_MODE").MustString("disable")
	DbCfg.Path = sec.Key("PATH").MustString(filepath.Join(setting.AppDataPath, "gitea.db"))
	DbCfg.Timeout = sec.Key("SQLITE_TIMEOUT").MustInt(500)
}

// parsePostgreSQLHostPort parses given input in various forms defined in
// https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING
// and returns proper host and port number.
func parsePostgreSQLHostPort(info string) (string, string) {
	host, port := "127.0.0.1", "5432"
	if strings.Contains(info, ":") && !strings.HasSuffix(info, "]") {
		idx := strings.LastIndex(info, ":")
		host = info[:idx]
		port = info[idx+1:]
	} else if len(info) > 0 {
		host = info
	}
	return host, port
}

func getPostgreSQLConnectionString(DBHost, DBUser, DBPasswd, DBName, DBParam, DBSSLMode string) (connStr string) {
	host, port := parsePostgreSQLHostPort(DBHost)
	if host[0] == '/' { // looks like a unix socket
		connStr = fmt.Sprintf("postgres://%s:%s@:%s/%s%ssslmode=%s&host=%s",
			url.PathEscape(DBUser), url.PathEscape(DBPasswd), port, DBName, DBParam, DBSSLMode, host)
	} else {
		connStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s%ssslmode=%s",
			url.PathEscape(DBUser), url.PathEscape(DBPasswd), host, port, DBName, DBParam, DBSSLMode)
	}
	return
}

// ParseMSSQLHostPort splits the host into host and port
func ParseMSSQLHostPort(info string) (string, string) {
	host, port := "127.0.0.1", "1433"
	if strings.Contains(info, ":") {
		host = strings.Split(info, ":")[0]
		port = strings.Split(info, ":")[1]
	} else if strings.Contains(info, ",") {
		host = strings.Split(info, ",")[0]
		port = strings.TrimSpace(strings.Split(info, ",")[1])
	} else if len(info) > 0 {
		host = info
	}
	return host, port
}

func getEngine() (*xorm.Engine, error) {
	connStr := ""
	var Param = "?"
	if strings.Contains(DbCfg.Name, Param) {
		Param = "&"
	}
	switch DbCfg.Type {
	case "mysql":
		connType := "tcp"
		if DbCfg.Host[0] == '/' { // looks like a unix socket
			connType = "unix"
		}
		tls := DbCfg.SSLMode
		if tls == "disable" { // allow (Postgres-inspired) default value to work in MySQL
			tls = "false"
		}
		connStr = fmt.Sprintf("%s:%s@%s(%s)/%s%scharset=utf8&parseTime=true&tls=%s",
			DbCfg.User, DbCfg.Passwd, connType, DbCfg.Host, DbCfg.Name, Param, tls)
	case "postgres":
		connStr = getPostgreSQLConnectionString(DbCfg.Host, DbCfg.User, DbCfg.Passwd, DbCfg.Name, Param, DbCfg.SSLMode)
	case "mssql":
		host, port := ParseMSSQLHostPort(DbCfg.Host)
		connStr = fmt.Sprintf("server=%s; port=%s; database=%s; user id=%s; password=%s;", host, port, DbCfg.Name, DbCfg.User, DbCfg.Passwd)
	case "sqlite3":
		if !EnableSQLite3 {
			return nil, errors.New("this binary version does not build support for SQLite3")
		}
		if err := os.MkdirAll(path.Dir(DbCfg.Path), os.ModePerm); err != nil {
			return nil, fmt.Errorf("Failed to create directories: %v", err)
		}
		connStr = fmt.Sprintf("file:%s?cache=shared&mode=rwc&_busy_timeout=%d", DbCfg.Path, DbCfg.Timeout)
	case "tidb":
		if !EnableTiDB {
			return nil, errors.New("this binary version does not build support for TiDB")
		}
		if err := os.MkdirAll(path.Dir(DbCfg.Path), os.ModePerm); err != nil {
			return nil, fmt.Errorf("Failed to create directories: %v", err)
		}
		connStr = "goleveldb://" + DbCfg.Path
	default:
		return nil, fmt.Errorf("Unknown database type: %s", DbCfg.Type)
	}

	return xorm.NewEngine(DbCfg.Type, connStr)
}

// NewTestEngine sets a new test xorm.Engine
func NewTestEngine(x *xorm.Engine) (err error) {
	x, err = getEngine()
	if err != nil {
		return fmt.Errorf("Connect to database: %v", err)
	}

	x.SetMapper(core.GonicMapper{})
	x.SetLogger(log.XORMLogger)
	x.ShowSQL(!setting.ProdMode)
	return x.StoreEngine("InnoDB").Sync2(tables...)
}

// SetEngine sets the xorm.Engine
func SetEngine() (err error) {
	x, err = getEngine()
	if err != nil {
		return fmt.Errorf("Failed to connect to database: %v", err)
	}

	x.SetMapper(core.GonicMapper{})
	// WARNING: for serv command, MUST remove the output to os.stdout,
	// so use log file to instead print to stdout.
	x.SetLogger(log.XORMLogger)
	x.ShowSQL(setting.LogSQL)
	return nil
}

// NewEngine initializes a new xorm.Engine
func NewEngine(migrateFunc func(*xorm.Engine) error) (err error) {
	if err = SetEngine(); err != nil {
		return err
	}

	if err = x.Ping(); err != nil {
		return err
	}

	if err = migrateFunc(x); err != nil {
		return fmt.Errorf("migrate: %v", err)
	}

	if err = x.StoreEngine("InnoDB").Sync2(tables...); err != nil {
		return fmt.Errorf("sync database struct error: %v", err)
	}

	return nil
}

// SyncDBStructs will sync database structs
func SyncDBStructs() error {
	if err := x.StoreEngine("InnoDB").Sync2(tables...); err != nil {
		return fmt.Errorf("sync database struct error: %v", err)
	}

	return nil
}

// Statistic contains the database statistics
type Statistic struct {
	Counter struct {
		User, Org, PublicKey,
		Repo, Watch, Star, Action, Access,
		Issue, Comment, Oauth, Follow,
		Mirror, Release, LoginSource, Webhook,
		Milestone, Label, HookTask,
		Team, UpdateTask, Attachment int64
	}
}

// GetStatistic returns the database statistics
func GetStatistic() (stats Statistic) {
	stats.Counter.User = CountUsers()
	stats.Counter.Org = CountOrganizations()
	stats.Counter.PublicKey, _ = x.Count(new(PublicKey))
	stats.Counter.Repo = CountRepositories(true)
	stats.Counter.Watch, _ = x.Count(new(Watch))
	stats.Counter.Star, _ = x.Count(new(Star))
	stats.Counter.Action, _ = x.Count(new(Action))
	stats.Counter.Access, _ = x.Count(new(Access))
	stats.Counter.Issue, _ = x.Count(new(Issue))
	stats.Counter.Comment, _ = x.Count(new(Comment))
	stats.Counter.Oauth = 0
	stats.Counter.Follow, _ = x.Count(new(Follow))
	stats.Counter.Mirror, _ = x.Count(new(Mirror))
	stats.Counter.Release, _ = x.Count(new(Release))
	stats.Counter.LoginSource = CountLoginSources()
	stats.Counter.Webhook, _ = x.Count(new(Webhook))
	stats.Counter.Milestone, _ = x.Count(new(Milestone))
	stats.Counter.Label, _ = x.Count(new(Label))
	stats.Counter.HookTask, _ = x.Count(new(HookTask))
	stats.Counter.Team, _ = x.Count(new(Team))
	stats.Counter.Attachment, _ = x.Count(new(Attachment))
	return
}

// Ping tests if database is alive
func Ping() error {
	if x != nil {
		return x.Ping()
	}
	return errors.New("database not configured")
}

// DumpDatabase dumps all data from database according the special database SQL syntax to file system.
func DumpDatabase(filePath string, dbType string) error {
	var tbs []*core.Table
	for _, t := range tables {
		t := x.TableInfo(t)
		t.Table.Name = t.Name
		tbs = append(tbs, t.Table)
	}
	if len(dbType) > 0 {
		return x.DumpTablesToFile(tbs, filePath, core.DbType(dbType))
	}
	return x.DumpTablesToFile(tbs, filePath)
}

// DumpDatabaseFixtures dumps all data from database to fixtures files on dirPath
func DumpDatabaseFixtures(dirPath string) error {
	for _, t := range tables {
		if err := dumpTableFixtures(t, dirPath); err != nil {
			return err
		}
	}
	return nil
}

func dumpTableFixtures(bean interface{}, dirPath string) error {
	table := x.TableInfo(bean)
	f, err := os.Create(filepath.Join(dirPath, table.Name+".yml"))
	if err != nil {
		return err
	}
	defer f.Close()

	const bufferSize = 100
	var start = 0
	for {
		objs, err := x.Table(table.Name).Limit(bufferSize, start).QueryInterface()
		if err != nil {
			return err
		}
		if len(objs) == 0 {
			break
		}

		data, err := yaml.Marshal(objs)
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		if err != nil {
			return err
		}
		if len(objs) < bufferSize {
			break
		}
		start += len(objs)
	}

	return nil
}

// RestoreDatabaseFixtures restores all data from dir to database
func RestoreDatabaseFixtures(dirPath string) error {
	for _, t := range tables {
		if err := restoreTableFixtures(t, dirPath); err != nil {
			return err
		}
	}
	return nil
}

func restoreTableFixtures(bean interface{}, dirPath string) error {
	table := x.TableInfo(bean)
	data, err := ioutil.ReadFile(filepath.Join(dirPath, table.Name+".yml"))
	if err != nil {
		return err
	}

	const bufferSize = 100
	var records = make([]map[string]interface{}, 0, bufferSize*10)
	err = yaml.Unmarshal(data, &records)
	if err != nil {
		return err
	}

	if len(records) == 0 {
		return nil
	}

	var columns = make([]string, 0, len(records[0]))
	for k := range records[0] {
		columns = append(columns, k)
	}
	sort.Strings(columns)

	qm := strings.Repeat("?,", len(columns))
	qm = "(" + qm[:len(qm)-1] + ")"

	_, err = x.Exec("DELETE FROM `" + table.Name + "`")
	if err != nil {
		return err
	}

	var sql = "INSERT INTO `" + table.Name + "` (`" + strings.Join(columns, "`,`") + "`) VALUES "
	var args = make([]interface{}, 0, bufferSize)
	var insertSQLs = make([]string, 0, bufferSize)
	for i, vals := range records {
		insertSQLs = append(insertSQLs, qm)
		for _, colName := range columns {
			args = append(args, vals[colName])
		}

		if i+1%100 == 0 || i == len(records)-1 {
			_, err = x.Exec(sql+strings.Join(insertSQLs, ","), args...)
			if err != nil {
				return err
			}
			insertSQLs = make([]string, 0, bufferSize)
			args = make([]interface{}, 0, bufferSize)
		}
	}

	return err
}
