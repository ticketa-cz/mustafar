
// ================== SETUP API ==================== //

package main

import (
	"fmt"
    "database/sql"
    _ "github.com/lib/pq"

    "log"
	"net/http"
    "github.com/gorilla/mux"
    "bufio"
    "os"

    "time"
    
    "encoding/json"
    "encoding/base64"
)

const (
    DB_HOST         = "172.19.0.2"
    DB_PORT         = "5432"
    DB_USER         = "mustafar"
    DB_PASSWORD     = "4eki2515"
    DB_NAME         = "ticketa"

    data_example_file = "adventure-time-time-sandwich.png"
    data_example_key = "sd3409sdl"
    data_expiration_days = 30
)

type Data struct {
    Data_key        string      `json:"data_key"`
    Data_value      string      `json:"data_value"`
    Data_expiration time.Time   `json:"data_expiration"`
}
type Datas []Data

type JsonResponse struct {
    Type            string      `json:"type"`
    Data            Data        `json:"data"`
    Message         string      `json:"message"`
    Error           error       `json:"error"`
}

func main() {

    // setup router //
    route_requests()
                    
}


// ================== SETUP FUNCTIONS ==================== //


// route requests //

func route_requests() {

    router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", link_basic)
    router.HandleFunc("/in/", link_insert)
    router.HandleFunc("/injson/", link_insert_json)
    router.HandleFunc("/out/{data_key}", link_output)
    router.HandleFunc("/outjson/", link_output_json)
    router.HandleFunc("/setup/", link_setup)
	log.Fatal(http.ListenAndServe(":8080", router))

}

// setup database //

func setupDB() *sql.DB {

    dbinfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME)
    db, _ := sql.Open("postgres", dbinfo)

    //checkErr(w, err)

    return db
}


// ================== BASIC FUNCTIONS ==================== //

func checkErr(w http.ResponseWriter, err error ) {
    
    if err != nil {
        var response = JsonResponse{}
        response = JsonResponse { Type: "Error", Error: err }
        json.NewEncoder(w).Encode(response)
    }
}

func printMessage( w http.ResponseWriter, message string ) {
    fmt.Fprintf(w, "\n")
    fmt.Fprintf(w, message)
    fmt.Fprintf(w, "\n")
}


// ================== REQUEST HANDLERS ==================== //

// basic url response //

func link_basic( w http.ResponseWriter, r *http.Request ) {

	printMessage( w, "Hello, nothing here!" )

}

// setup url response //

func link_setup( w http.ResponseWriter, r *http.Request ) {
	
    printMessage( w, "Setting up the database...")

    // setup db //
    db := setupDB()
    _, drop := db.Exec(`DROP TABLE IF EXISTS APP_DATA;`)
    checkErr(w, drop)
    _, create := db.Exec(`CREATE TABLE APP_DATA (ID SERIAL PRIMARY KEY NOT NULL, DATA_KEY text, DATA_VALUE bytea, DATA_EXPIRATION text);`)
    checkErr(w, create)
    
}


// data insert function //

func link_insert( w http.ResponseWriter, r *http.Request ) {

    w.Header().Set("Content-Type", "application/json; charset=UTF-8")
    w.WriteHeader(http.StatusOK)
    var response = JsonResponse{}
    var data_output = Data{}

    // open image file //

    imgFile, err := os.Open(data_example_file)
    if err != nil {
        os.Exit(1)
    }

    defer imgFile.Close()

    // create a new buffer base on file size //

    fInfo, _ := imgFile.Stat()
    var size int64 = fInfo.Size()
    buf := make([]byte, size)

    // read file content into buffer //

    fReader := bufio.NewReader(imgFile)
    fReader.Read(buf)

    // connect db //

    db := setupDB()

    // prepare data //

    data_encoded     := base64.StdEncoding.EncodeToString(buf)
    today_date       := time.Now()
    data_expiration  := today_date.AddDate(0, 0, data_expiration_days)
    data_expirationU := data_expiration.Unix()

    // insert data //

    _, err = db.Exec("INSERT INTO app_data (data_key, data_value, data_expiration) VALUES ($1, $2, $3)", data_example_key, data_encoded, data_expirationU)

    // check errors //

    checkErr(w, err)

    // response //

    data_output = Data { Data_key: data_example_key, Data_value: data_encoded, Data_expiration: data_expiration }
    response = JsonResponse { Type: "Success", Message: "The data has been inserted successfully!", Data: data_output }

    json.NewEncoder(w).Encode(response)

}

// data output function //

func link_output( w http.ResponseWriter, r *http.Request ) {

    w.Header().Set("Content-Type", "application/json; charset=UTF-8")
    w.WriteHeader(http.StatusOK)
    var response = JsonResponse{}

    id_passed := mux.Vars(r)["data_key"]

    if id_passed == "" {

        response = JsonResponse { Type: "Error", Message: "You are missing Data ID parameter." }

    } else {

        // connect db //
        
        db := setupDB()

        rows, err := db.Query("SELECT id, data_key, data_value, data_expiration FROM app_data WHERE data_key = $1", id_passed)
        checkErr(w, err)
        defer rows.Close()

        today_date  := time.Now()
        date_newest := time.Now()
        var data_output = Data{}

        // loop through rows //
        
        for rows.Next() {

            var id                  int
            var data_key            string
            var data_value          string
            var data_expiration     int64
    
            err = rows.Scan( &id, &data_key, &data_value, &data_expiration )

            data_expiration_time := time.Unix(data_expiration, 0)
    
            // check errors //

            checkErr(w, err)

            // get the newest record // and before expiration //

            if data_expiration_time.After( date_newest ) && data_expiration_time.After( today_date ) {

                data_output = Data { Data_key: data_key, Data_value: data_value, Data_expiration: data_expiration_time }
                date_newest = data_expiration_time

            }

            // erase record after expiration //
            
            if data_expiration_time.Before( today_date ) {
                
                _, del_err := db.Exec("DELETE FROM app_data WHERE id = $1", id)
                checkErr(w, del_err)

            }
            
        }

        // check row errors //

        err = rows.Err()
        checkErr(w, err)

        // response //
        
        response = JsonResponse { Type: "Success", Data: data_output, Message: "Data key parameter value passed: " + id_passed }

    }

    json.NewEncoder(w).Encode(response)

}


///////////////////////// JSON VERSIONS //////////////////////////


// link insert from JSON //

func link_insert_json( w http.ResponseWriter, r *http.Request ) {

    w.Header().Set("Content-Type", "application/json; charset=UTF-8")
    w.WriteHeader(http.StatusOK)

    var input Data
    var response = JsonResponse{}
    var data_output = Data{}

    // decode JSON input //

    err := json.NewDecoder(r.Body).Decode(&input)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // open image file //

    imgFile, err := os.Open(input.Data_value)
    if err != nil {
        os.Exit(1)
    }

    defer imgFile.Close()

    // create a new buffer base on file size //

    fInfo, _ := imgFile.Stat()
    var size int64 = fInfo.Size()
    buf := make([]byte, size)

    // read file content into buffer //

    fReader := bufio.NewReader(imgFile)
    fReader.Read(buf)

    // connect db //

    db := setupDB()

    // prepare data //

    data_encoded     := base64.StdEncoding.EncodeToString(buf)
    today_date       := time.Now()
    data_expiration  := today_date.AddDate(0, 0, data_expiration_days)
    data_expirationU := data_expiration.Unix()

    // insert data //

    _, err = db.Exec("INSERT INTO app_data (data_key, data_value, data_expiration) VALUES ($1, $2, $3)", input.Data_key, data_encoded, data_expirationU)

    // check errors //

    checkErr(w, err)

    // response //

    data_output = Data { Data_key: input.Data_key, Data_value: data_encoded, Data_expiration: data_expiration }
    response = JsonResponse { Type: "Success", Message: "The data has been inserted successfully!", Data: data_output }

    json.NewEncoder(w).Encode(response)

}

// data output function //

func link_output_json( w http.ResponseWriter, r *http.Request ) {

    w.Header().Set("Content-Type", "application/json; charset=UTF-8")
    w.WriteHeader(http.StatusOK)
    var response = JsonResponse{}
    var input Data

    // decode JSON input //

    err := json.NewDecoder(r.Body).Decode(&input)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    if input.Data_key == "" {

        response = JsonResponse { Type: "Error", Message: "You are missing data key parameter." }

    } else {

        // connect db //
        
        db := setupDB()

        rows, err := db.Query("SELECT id, data_key, data_value, data_expiration FROM app_data WHERE data_key = $1", input.Data_key)
        checkErr(w, err)
        defer rows.Close()

        today_date  := time.Now()
        date_newest := time.Now()
        var data_output = Data{}

        // loop through rows //
        
        for rows.Next() {

            var id                  int
            var data_key            string
            var data_value          string
            var data_expiration     int64
    
            err = rows.Scan( &id, &data_key, &data_value, &data_expiration )

            data_expiration_time := time.Unix(data_expiration, 0)
    
            // check errors //

            checkErr(w, err)

            // get the newest record // and before expiration //

            if data_expiration_time.After( date_newest ) && data_expiration_time.After( today_date ) {

                data_output = Data { Data_key: data_key, Data_value: data_value, Data_expiration: data_expiration_time }
                date_newest = data_expiration_time

            }

            // erase record after expiration //
            
            if data_expiration_time.Before( today_date ) {
                
                _, del_err := db.Exec("DELETE FROM app_data WHERE id = $1", id)
                checkErr(w, del_err)

            }
            
        }

        // check row errors //

        err = rows.Err()
        checkErr(w, err)

        // response //
        
        response = JsonResponse { Type: "Success", Data: data_output, Message: "Data key parameter value passed: " + input.Data_key }

    }

    json.NewEncoder(w).Encode(response)

}