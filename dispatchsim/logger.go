package dispatchsim

import (
	"flag"
	"go/build"
	"log"
	"os"
)

var (
	Log *log.Logger
)

// Intialize the logger for dispatcher to log its data to the .log file
func initLogger() {
	// set location of log file
	var logpath = build.Default.GOPATH + "/src/github.com/harrizontal/dispatchserver/assets/driver/dispatcher.log"

	flag.Parse()
	var file, err1 = os.Create(logpath)

	if err1 != nil {
		panic(err1)
	}

	Log = log.New(file, "", log.LstdFlags|log.Lshortfile)
	Log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))

	Log.Println("LogFile : " + logpath)
}
