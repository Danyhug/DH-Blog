package config

type Server struct {
	Address   string
	HttpPort  int
	HttpsPort int
	CertFile  string
	KeyFile   string
}

type DataBase struct {
	Type   string
	DBFile string // For SQLite, this will be the file path
	Dsn    string // For other databases like MySQL, this will be the DSN
}

type Config struct {
	Server    Server   `json:"server"`
	DataBase  DataBase `json:"database"`
	JwtSecret string
}

func DefaultConfig() *Config {
	return &Config{
		Server: Server{
			Address:   "0.0.0.0",
			HttpPort:  2233,
			HttpsPort: -1,
		},
		DataBase: DataBase{
			Type:   "sqlite3",
			DBFile: "dhblog.db", // Default SQLite database file
			Dsn:    "",          // DSN will be empty for SQLite by default
		},
		JwtSecret: "test",
	}
}
