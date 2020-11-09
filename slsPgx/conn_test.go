package slsPgx

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"reflect"
	"testing"
	"time"
)

const (
	connectionString = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
)

func createMockClients(n int) []*pgx.Conn {
	var clients []*pgx.Conn
	for i := 0; i < n; i++ {
		c, err := pgx.Connect(context.Background(), connectionString)
		if err != nil {
			continue
		}
		clients = append(clients, c)
	}

	return clients
}

func cleanMockClients(clients []*pgx.Conn) {
	for _, c := range clients {
		_ = c.Close(context.Background())
	}
}

func closeConnectionAsync(client *pgx.Conn, delaySec int) {
	time.Sleep(time.Duration(delaySec) * time.Second)
	client.Close(context.Background())
	fmt.Println("connection closed...", client.IsClosed())
}

func Test_slsConn_Connect(t *testing.T) {
	type fields struct {
		Config   SlsConnConfigParams
		conn     *pgx.Conn
		connCred connCred
	}
	type args struct {
		ctx              context.Context
		connectionString string
		numOfClients     int
	}
	tests := []struct {
		name             string
		fields           fields
		args             args
		shouldCloseAsync bool
		want             *pgx.Conn
		wantErr          bool
	}{
		{
			name: "Client should connect normally",
			fields: fields{
				Config: SlsConnConfigParams{
					Debug: Bool(true),
				},
			},
			args: args{
				ctx:              context.Background(),
				connectionString: connectionString,
				numOfClients:     1,
			},
			shouldCloseAsync: false,
			want:             &pgx.Conn{},
			wantErr:          false,
		},
		{
			name: "Client should be able to reconnect",
			fields: fields{
				Config: SlsConnConfigParams{
					BackoffMaxRetries: Int(5),
					Debug:             Bool(true),
				},
			},
			args: args{
				ctx:              context.Background(),
				connectionString: connectionString,
				numOfClients:     100,
			},
			shouldCloseAsync: true,
			want:             &pgx.Conn{},
			wantErr:          false,
		},
		{
			name: "Client should not be able to reconnect after n attempt",
			fields: fields{
				Config: SlsConnConfigParams{
					BackoffMaxRetries: Int(3),
					Debug:             Bool(true),
				},
			},
			args: args{
				ctx:              context.Background(),
				connectionString: connectionString,
				numOfClients:     100,
			},
			shouldCloseAsync: false,
			want:             nil,
			wantErr:          true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClients := createMockClients(tt.args.numOfClients)
			if tt.shouldCloseAsync {
				go closeConnectionAsync(mockClients[0], 1)
			}

			s := New(tt.fields.Config)
			_, err := s.Connect(tt.args.ctx, tt.args.connectionString)
			if (err != nil) != tt.wantErr {
				t.Errorf("Connect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			cleanMockClients(mockClients)
		})
	}
}

func Test_slsConn_getIdleProcessesListByMinimumTimeout(t *testing.T) {
	type fields struct {
		Config   slsConnConfig
		conn     *pgx.Conn
		connCred connCred
	}
	type args struct {
		ctx          context.Context
		numOfClients int
		sleepTimeSec float32
		wakeUpClient bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "should correctly get idle process list ordered by date",
			fields: fields{
				Config:   slsConnConfig{},
				conn:     nil,
				connCred: connCred{},
			},
			args: args{
				ctx:          context.Background(),
				numOfClients: 10,
				sleepTimeSec: 1,
				wakeUpClient: true,
			},
			want:    9,
			wantErr: false,
		},

		{
			name: "should correctly get not idle process",
			fields: fields{
				Config:   slsConnConfig{},
				conn:     nil,
				connCred: connCred{},
			},
			args: args{
				ctx:          context.Background(),
				numOfClients: 0,
				sleepTimeSec: 1,
				wakeUpClient: false,
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "should correctly get not idle process, clients have not been idling for enough time",
			fields: fields{
				Config:   slsConnConfig{},
				conn:     nil,
				connCred: connCred{},
			},
			args: args{
				ctx:          context.Background(),
				numOfClients: 10,
				sleepTimeSec: 0.4,
				wakeUpClient: false,
			},
			want:    0,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClients := createMockClients(tt.args.numOfClients)
			time.Sleep(time.Duration(tt.args.sleepTimeSec) * time.Second)
			// Wake up one client
			if tt.args.wakeUpClient {
				if _, err := mockClients[0].Query(tt.args.ctx, "SELECT 1+1 AS result"); err != nil {
					t.Error("Test failed: ", err)
					return
				}
			}

			s := New(SlsConnConfigParams{})
			if _, err := s.Connect(tt.args.ctx, connectionString); err != nil {
				t.Error("Test failed: ", err)
				return
			}
			got, err := s.getIdleProcessesListByMinimumTimeout(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("getIdleProcessesListOrderByDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(len(got), tt.want) {
				t.Errorf("getIdleProcessesListOrderByDate() got = %v, want %v", len(got), tt.want)
				return
			}

			cleanMockClients(mockClients)
			if err := s.Close(tt.args.ctx); err != nil {
				t.Error("Test failed: ", err)
				return
			}
		})
	}
}

func TestSlsConn_getProcessCount(t *testing.T) {
	type args struct {
		numClients int
	}
	tests := []struct {
		name    string
		args    args
		want    uint8
		wantErr bool
	}{
		{
			name:    "should get 4 connected clients",
			args:    args{numClients: 3},
			want:    4,
			wantErr: false,
		},
		{
			name:    "should get 100 connected clients",
			args:    args{numClients: 200},
			want:    100,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(SlsConnConfigParams{})
			if _, err := c.Connect(context.Background(), connectionString); err != nil {
				t.Errorf("getProcessCount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			mockClients := createMockClients(tt.args.numClients)

			got, err := c.getProcessCount(context.Background())

			cleanMockClients(mockClients)
			if err := c.Close(context.Background()); err != nil {
				t.Error("Test failed: ", err)
				return
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("getProcessCount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getProcessCount() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_slsConn_parseURL(t *testing.T) {
	type args struct {
		connString string
	}
	type want struct {
		user string
		database string
	}
	tests := []struct {
		name    string
		args    args
		want    want
		wantErr bool
	}{
		{
			name:    "Should parse a correct url",
			args:    args{connString: connectionString},
			want: want{
				user:     "postgres",
				database: "postgres",
			},
			wantErr: false,
		},
		{
			name:    "Should parse a correct url",
			args:    args{connString: "postgres://matteo:mypass@pg–instance1.123456789012.us-east-1.rds.amazonaws.com:5432/posts?sslmode=disable"},
			want: want{
				user:     "matteo",
				database: "posts",
			},
			wantErr: false,
		},
		{
			name:    "Should parse an incorrect url",
			args:    args{connString: "https://someurl.com"},
			want: want{
				user:     "",
				database: "",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(SlsConnConfigParams{})

			if err := s.parseURL(tt.args.connString); (err != nil) != tt.wantErr {
				t.Errorf("parseURL() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			w := want{
				user:     s.connCred.user,
				database: s.connCred.database,
			}

			if !reflect.DeepEqual(w, tt.want) {
				t.Errorf("parseUrl() got = %v, want %v", w, tt.want)
			}
		})
	}
}

func TestSlsConn_Query(t *testing.T) {
	tests := []struct {
		name    string
		want    int
		wantErr bool
	}{
		{
			name:    "Should query successfully event though the connection was killed",
			want:    2,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s1 := New(SlsConnConfigParams{})
			s2 := New(SlsConnConfigParams{
				Debug: Bool(true),
			})
			if _, err := s1.Connect(context.Background(), connectionString); err != nil {
				t.Error("Test failed: ", err)
				return
			}
			if _, err := s2.Connect(context.Background(), connectionString); err != nil {
				t.Error("Test failed: ", err)
				return
			}
			pid := int(s2.GetConnection().PgConn().PID())
			// Kill the mock client connections and try to query
			if err := s1.killProcesses(context.Background(), []int{pid}); err != nil {
				t.Error("Could not kill process: ", err)
				return
			}
			
			var res int
			rows, err := s2.Query(context.Background(), "SELECT 1+1 AS result")
			if (err != nil) != tt.wantErr {
				t.Errorf("Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			for rows.Next() {
				if err := rows.Scan(&res); err != nil {
					return
				}
			}

			if !reflect.DeepEqual(res, tt.want) {
				t.Errorf("Query() got = %v, want %v", res, tt.want)
			}
		})
	}
}

func TestSlsConn_Exec(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{
			name:    "Should query successfully event though the connection was killed",
			want:    "SELECT 1",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s1 := New(SlsConnConfigParams{})
			s2 := New(SlsConnConfigParams{
				Debug: Bool(true),
			})
			if _, err := s1.Connect(context.Background(), connectionString); err != nil {
				t.Error("Test failed: ", err)
				return
			}
			if _, err := s2.Connect(context.Background(), connectionString); err != nil {
				t.Error("Test failed: ", err)
				return
			}
			pid := int(s2.GetConnection().PgConn().PID())
			// Kill the mock client connections and try to query
			if err := s1.killProcesses(context.Background(), []int{pid}); err != nil {
				t.Error("Could not kill process: ", err)
				return
			}
			
			got, err := s2.Exec(context.Background(), "SELECT 1+1 AS result")
			if (err != nil) != tt.wantErr {
				t.Errorf("Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.String(), tt.want) {
				t.Errorf("Query() got = %v, want %v", got.String(), tt.want)
			}
		})
	}
}
