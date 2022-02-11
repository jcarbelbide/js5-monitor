package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"js5-monitor/js5connection"
	"log"
	"net/http"
	"time"
)

type ServerResetInfo struct {
	LastResetTime     time.Time `json:"reset_time"`
	LastResetTimeUnix int64     `json:"last_reset_time_unix"`
	LastServerUptime  int64     `json:"last_server_uptime"`
}

var LastServerResetInfo *ServerResetInfo
var db *sql.DB

func main() {
	// Init Logger
	logFile := initLogging()
	defer logFile.Close()

	// Database
	db = initDatabase()

	// Start JS5 Monitor
	go MonitorJS5()

	// Init Router
	r := mux.NewRouter()

	// Route Handlers / Endpoints
	r.HandleFunc("/lastreset", getLastReset).Methods("GET")

	log.Fatal(http.ListenAndServe(":8081", r))
}

// -------------------------------------------------------------------------- //
// ---------------------------- Request Handlers ---------------------------- //
// -------------------------------------------------------------------------- //

func getLastReset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	lastReset := LastServerResetInfo
	json.NewEncoder(w).Encode(lastReset)
	return
}

func consoleLoop() {

	js5, err := js5connection.New()
	fmt.Println(err)
	//fmt.Println(js5.Ping())

	var loopCounter float64 = 0
	for {
		resp, err := js5.Ping()
		elapsedTime := js5connection.PingInterval.Seconds() * loopCounter
		fmt.Println("Resp ", resp[:10], " |  Error: ", err, " |  Seconds Elapsed: ", elapsedTime)
		if err != nil {
			fmt.Println("Connection broken! at " + time.Now().String())
			break
		}
		time.Sleep(js5connection.PingInterval)
		loopCounter++
	}

}

// MonitorJS5
//  1. Start by initializing db and getting the first server reset. If it's the first time, just set time to now.
//	2. Initialize JS5 Connection. If it fails to connect, keep trying. Server may be down.
//		3. Ping the connection every 5 seconds in a new loop.
//		4. If the connection breaks, break out of current loop
//	5. Be sure to take the difference between the time for last reset and current reset.
//	6. Write to DB
//	7. Set LastServerResetInfo to current one.
//	8. restart loop at step 2.
func MonitorJS5() {

	LastServerResetInfo = initServerResetInfo(db)

	for { // Infinite Loop
		time.Sleep(js5connection.PingInterval)
		js5, err := js5connection.New()
		fmt.Println(&js5)

		if err != nil {
			// Start over by trying a new connection
			continue
		}

		for { // Pinging Loop. Ping until connection drops
			_, err := js5.Ping()
			if err != nil {
				break
			}

			// For testing:
			//if requestKeyboardInput() {break}

			time.Sleep(js5connection.PingInterval)
		}

		currentServerResetInfo := ServerResetInfo{
			LastResetTime:     time.Now(),
			LastResetTimeUnix: time.Now().Unix(),
			LastServerUptime:  time.Now().Unix() - LastServerResetInfo.LastResetTimeUnix,
		}

		addNewServerResetInfo(currentServerResetInfo, db)

		LastServerResetInfo = &currentServerResetInfo
	}
}

func initServerResetInfo(database *sql.DB) *ServerResetInfo {

	var lastServerReset ServerResetInfo

	fmt.Println("Looking for entry in database...")
	entryExists, lastServerReset := queryDBForLastServerReset(database)

	if !entryExists { // entry does not exist in db
		lastServerReset = ServerResetInfo{
			LastResetTime:     time.Now(),
			LastResetTimeUnix: time.Now().Unix(),
			LastServerUptime:  0,
		}
		fmt.Println("Could not find entry in database")
		addNewServerResetInfo(lastServerReset, db)
	}

	return &lastServerReset

}

func requestKeyboardInput() bool {
	var input string
	fmt.Scanln(&input)
	return input == "y"
}
