/*
Created by: Alexandre Vanham
Modified by: Alexandre Vanham
Date: 21-11-2017
Version: 0
*/
package main

import (
	"net/http"
	"github.com/gorilla/mux"
	"encoding/json"
	"log"
	"fmt"
	"os"
	"encoding/base64"
	"encoding/hex"
	"time"
	"path/filepath"
	"io/ioutil"
	"html/template"

	"database/sql"
	_ "github.com/denisenkom/go-mssqldb"
	"strconv"
	"github.com/auth0/go-jwt-middleware"
	"github.com/dgrijalva/jwt-go"
)



//----------------------------------Variables et struct----------------------------

//définition de la structure du fichier de configuration
type configStruct struct {
	Database struct {
		Server     string `json:"server"`
		DbName    string `json:"dbName"`
		User     string `json:"user"`
		Password string `json:"password"`
		Port     string `json:"port"`
	} `json:"database"`
	Host string `json:"host"`
	Port string `json:"port"`
	DirPath string `json:"dirPath"`
}

//définition de la structure d'un document
type iDoc struct{
	Id_Code string `json:"idCode"`
	Id_Guid string `json:"idGuid"`
	Idt_Code string `json:"idtCode"`
	Id_Comment string `json:"idComment"`
	Id_Path string `json:"idPath"`
	Id_FileName string `json:"idFileName"`
	Id_Classed string `json:"idClassed"`
	Id_Creation_Date string `json:"idCreationDate"`
	Id_Version string `json:"idVersion"`
	Id_Size string `json:"idSize"`
	Id_JSon string `json:"idJSon"`
	Id_Binary string `json:"idBinary"`
	Id_BinaryType string `json:"idBinaryType"`
	Id_BinaryLg string `json:"idBinaryLg"`
	UCreate string `json:"uCreate"`
	DCreate string `json:"dCreate"`
	UUpdate string `json:"uUpdate"`
	DUpdate string `json:"dUpdate"`
}

//définition de la structure d'un user
type uDatas struct {
	Us_Code string `json:"codeUser"`
	Us_Last_Name string `json:"nomUser"`
	Us_First_Name string `json:"prenomUser"`
	Us_Password string `json:"password"`
}

var mySigningKey = []byte("secret")
var Config, _ = loadConfiguration("config.json")








//---------------------------------- Handlers ----------------------------

// envoye un token signé
func GetTokenHandler(w http.ResponseWriter, r *http.Request){

	vars := mux.Vars(r)
	userCode := vars["userCode"]
	log.Println("---func GetTokenHandler: user : ---" + userCode)

	querySt := "exec [iDoc].[GetUser] @UsCode = '" + userCode + "'"
	
	sqlData := querySqlUser(querySt)
	log.Println("----------Retour query: func getDoc(sqlData): ---------", sqlData)

	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
    claims["admin"] = true
    completName := sqlData.Us_Last_Name + sqlData.Us_First_Name
    claims["name"] = completName
    claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

    tokenString, _ := token.SignedString(mySigningKey)
    w.Write([]byte(tokenString))
    log.Println("---token sent : ----"/* + tokenString*/)
}


//retourne un document (json) en fonction d'un guid passé; exple guid: C61E7502-1D7A-4F4E-9577-CE73CE76D0E4
var getDoc = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	guid := vars["guid"]

	querySt := "exec iDoc.[GetDocByGuid] @idGuid = '" + guid + "'"
	
	sqlData := querySqlDoc(querySt)
	log.Println("----------Retour query: func getDoc(sqlData.Id_FileName): ---------", sqlData[0].Id_FileName)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(sqlData); err != nil {
		log.Println(err)
	}

	return
})

//enregistre un document dans l'intradoc, reçoit un json
var uploadDoc = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	var document iDoc

	//read body request
	if err := json.NewDecoder(r.Body).Decode(&document); err != nil {
		log.Println(err)
	}

	log.Println(document)

	//get datas from body request
	code := document.Id_Code
	typedoc := document.Idt_Code
	comment := document.Id_Comment
	path := Config.DirPath + document.Id_FileName
	nom := document.Id_FileName
	classed := document.Id_Classed
	modTime := document.Id_Creation_Date
	version := document.Id_Version
	size := document.Id_Size
	jsondoc := document.Id_JSon
	base64 := document.Id_Binary
	//log.Println("----------content base64: ---------", base64)
	hexad := base64Decode(base64)
	//log.Println("----------content binaire: ---------", hexad)
	strDoc := base64DecodeStr(base64)
	//log.Println("----------content strDoc: ---------", strDoc)
	binType := document.Id_BinaryType
	binLg := document.Id_BinaryLg

	modTimeDMY := modTime[0:10]

	querySt := "Declare @idCode int exec @idCode = iDoc.iDocUpsert @idCode = " + code + ", @idtCode = " + typedoc + ", @idComment	= '" + comment + "', @idPath = '" + path +"', @idFileName = '" + nom + "', @idClassed = " + classed + ", @idCreation_Date = '" + modTimeDMY + "', @idVersion = " + version + ", @idSize = " + size + ", @idJSon = '" + jsondoc + "', @idBinary = 0x" + hexad + ", @idBinaryType = '" + binType + "', @idBinaryLg = '" + binLg + "', @usCode = 'AV'"

	//executing sql query and getting n rows affected
	retour := execSql(querySt)
	log.Println("----------Retour query: func uploadDoc(sqlData): ---------", retour)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode("---Rows Affected--- : " + strconv.FormatInt(retour, 10)); err != nil {
		log.Println(err)
	}else{
		fmt.Fprintf(w, "document n° %s, named %s has been uploaded or modified in DB!", document.Id_Code, document.Id_FileName)
		log.Println("----------document uploaded:---", document.Id_FileName)
	}

	writeDocument(strDoc, nom, Config.DirPath)

	return
})

//retourne la liste documents non classé en fonction d'une personne; exple user : 
var waitingDoc = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	userCode := vars["userCode"]
	//log.Println("----------func waitingDoc: user : ---" + userCode)

	querySt := "Exec iDoc.[GetDocWaiting] @usCode = '" + userCode + "'"

	sqlData := querySqlDoc(querySt)
	log.Println("----------Retour query: func waitingDoc(sqlData): ---------", sqlData)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(sqlData); err != nil {
		log.Println(err)
	}
	return
})

//retourne une liste documents en fonction de paramètres passés (affaire,commentaires,...) exple : 
var listDoc = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	userCode := vars["userCode"]
	if(userCode == "_"){
		userCode = "%"
	}

	project := vars["project"]
	if(project == "_"){
		project = "%"
	}

	keywords := vars["keywords"]
	if(keywords == "_"){
		keywords = "%"
	}

	dateCreation := vars["dateCreation"]
	if(dateCreation == "_"){
		dateCreation = ""
	}else{
		dateCreation = dateCreation[0:10]
	}

	fmt.Println(keywords + userCode)

	querySt := "exec iDoc.GetDocByLinkedCode @usCode = 'AV', @idltCode = 'PJ', @idltLinkedCode = '" + project + "', @idCreationDate = '" + dateCreation + "'"

	sqlData := querySqlDoc(querySt)
	log.Println("----------Retour query: func listDoc(sqlData): ---------", sqlData)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(sqlData); err != nil {
		log.Println(err)
	}
	return
})

//supprime un document en fonction d'un guid passé; exple guid: 
var deleteDoc = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	
	vars := mux.Vars(r)
	guid := vars["guid"]

	querySt := "delete from iDoc.IDoc where id_Guid ='" + guid + "'" //-->procedure!?
	//log.Println(querySt)

	retour := execSql(querySt)
	log.Println("----------Retour query: func deleteDoc(sqlData): ---------", retour)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode("----------Rows Affected--- : " + strconv.FormatInt(retour, 10)); err != nil {
		log.Println(err)
	}
	return
})

func formHandler(w http.ResponseWriter, r *http.Request) {
    tmpl, err := template.ParseFiles("views/form_upload.html")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    checkErr(err, "--- func formHandler() template.ParseFiles()---")
    log.Println("--- func formHandler():  page form requested ---")

    tmpl.Execute(w, nil)    // Affiche le contenu de "views/form_upload.html"
}







//---------------------------------- Requetes SQL ----------------------------

//fonction qui retourne le result d'un query
func querySqlDoc(querySt string) map[int]iDoc {
	var (
		myDoc iDoc
		id_Code string
		id_Guid string
		idt_Code string
		id_Comment string
		id_Path string
		id_FileName string
		id_Classed string
		id_Creation_Date string
		id_Version string
		id_Size string
		id_JSon string
		id_Binary string
		id_BinaryType string
		id_BinaryLg string
		uCreate string
		dCreate string
		uUpdate string
		dUpdate string
		iCount int = 0
	)

	sqlDatas := make(map[int]iDoc)

	db, errdb := sql.Open("mssql", "server=" + Config.Database.Server +
										";database=" + Config.Database.DbName +
										";user id=" + Config.Database.User +
										";password=" + Config.Database.Password +
										";port=" + Config.Database.Port)
	if errdb != nil {
		log.Println("----------Error open db---:", errdb.Error())
	}else{
		log.Println("----------Connection opened ---------")
	}

	rows, err := db.Query(querySt)
	if err != nil {
		//log.Fatal(err)
		log.Println("----------Error querySqlDoc Query()---:", err.Error())
	}

	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&id_Code, &id_Guid, &idt_Code, &id_Comment, &id_Path, &id_FileName, &id_Classed, &id_Creation_Date, &id_Version, &id_Size, &id_JSon, &id_Binary, &id_BinaryType, &id_BinaryLg, &uCreate, &dCreate, &uUpdate, &dUpdate)
		if err != nil {
			//log.Fatal(err)
			log.Println("----------Error querySqlDoc rows.Scan()---:", err.Error())
		}else{
			if(len(id_Code) != 0 && id_Code != ""){
				myDoc.Id_Code = id_Code
			}
			if(len(id_Guid) != 0 && id_Guid != ""){
				myDoc.Id_Guid = id_Guid
			}
			if(len(idt_Code) != 0 && idt_Code != ""){
				myDoc.Idt_Code = idt_Code
			}
			if(len(id_Comment) != 0 && id_Comment != ""){
				myDoc.Id_Comment = id_Comment
			}
			if(len(id_Path) != 0 && id_Path != ""){
				myDoc.Id_Path = id_Path
			}
			if(len(id_FileName) != 0 && id_FileName != ""){
				myDoc.Id_FileName = id_FileName
			}
			if(len(id_Classed) != 0 && id_Classed != ""){
				myDoc.Id_Classed = id_Classed
			}
			if(len(id_Creation_Date) != 0 && id_Creation_Date != ""){
				myDoc.Id_Creation_Date = id_Creation_Date
			}
			if(len(id_Version) != 0 && id_Version != ""){
				myDoc.Id_Version = id_Version
			}
			if(len(id_Size) != 0 && id_Size != ""){
				myDoc.Id_Size = id_Size
			}
			if(len(id_JSon) != 0 && id_JSon != ""){
				myDoc.Id_JSon = id_JSon
			}
			if(len(id_Binary) != 0 && id_Binary != ""){
				dataDoc := readDocument(id_Path)
				if(dataDoc != "no document!"){
					myDoc.Id_Binary = dataDoc
				}else{
					myDoc.Id_Binary = id_Binary
				}
			}
			if(len(id_BinaryType) != 0 && id_BinaryType != ""){
				myDoc.Id_BinaryType = id_BinaryType
			}
			if(len(id_BinaryLg) != 0 && id_BinaryLg != ""){
				myDoc.Id_BinaryLg = id_BinaryLg
			}
			if(len(uCreate) != 0 && uCreate != ""){
				myDoc.UCreate = uCreate
			}
			if(len(dCreate) != 0 && dCreate != ""){
				myDoc.DCreate = dCreate
			}
			if(len(uUpdate) != 0 && uUpdate != ""){
				myDoc.UUpdate = uUpdate
			}
			if(len(dUpdate) != 0 && dUpdate != "" ){
				myDoc.DUpdate = dUpdate
			}
			sqlDatas[iCount] = myDoc
			iCount++
		}
	}
	//log.Println("----------Retour query: func querySqlDoc(sqlDatas): ---------", sqlDatas)

	defer db.Close()
	defer log.Println("----------Connection closed ---------")
	return sqlDatas
}

//fonction qui execute un query
func execSql(querySt string) int64 {
	db, errdb := sql.Open("mssql", "server=" + Config.Database.Server +
										";database=" + Config.Database.DbName +
										";user id=" + Config.Database.User +
										";password=" + Config.Database.Password +
										";port=" + Config.Database.Port)
	if errdb != nil {
		log.Println("----------Error open db---:", errdb.Error())
	}
	log.Println("----------Connection opened ---------")

	result, err := db.Exec(querySt)
	if err != nil {
		//log.Fatal(err)
		log.Println("---Error execSql Exec()---:", err.Error())
	}

	affect, err := result.RowsAffected()
	if err != nil {
		//log.Fatal(err)
		log.Println("----------Error execSql RowsAffected()---:", err.Error())
	}

	defer db.Close()
	defer log.Println("----------Connection closed ---------")
	return affect
}

//fonction qui retourne le result d'un query
func querySqlUser(querySt string) uDatas {
	var (
		userdatas uDatas
		us_Code string
		us_Last_Name string
		us_First_Name string
		//us_password string
	)


	db, errdb := sql.Open("mssql", "server=" + Config.Database.Server +
										";database=" + Config.Database.DbName +
										";user id=" + Config.Database.User +
										";password=" + Config.Database.Password +
										";port=" + Config.Database.Port)
	if errdb != nil {
		log.Println("----------Error open db---:", errdb.Error())
	}else{
		log.Println("----------Connection opened ---------")
	}

	rows, err := db.Query(querySt)
	if err != nil {
		log.Println("----------Error querySqlUser Query()---:", err.Error())
	}

	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&us_Code, &us_Last_Name, &us_First_Name)
		if err != nil {
			log.Println("---Error querySqlUser rows.Scan()---:", err.Error())
		}else{
			if(len(us_Code) != 0 && us_Code != ""){
				userdatas.Us_Code = us_Code
			}
			if(len(us_Last_Name) != 0 && us_Last_Name != ""){
				userdatas.Us_Last_Name = us_Last_Name
			}
			if(len(us_First_Name) != 0 && us_First_Name != ""){
				userdatas.Us_First_Name = us_First_Name
			}
		}
	}
	log.Println("----------Retour query: func querySqlUser(sqlDatas): ---------", userdatas)

	defer db.Close()
	defer log.Println("----------Connection closed ---------")
	return userdatas
}






//---------------------------------- Fonctions utilitaires et autres ----------------------------

func checkErr(err error, msg string) {
	if err != nil {
		log.Println("----------Error " + msg + ": ---", err)
	}
}

//hexa to base64
func base64Encode(hexa string) string {
	log.Println("---hexa---:", hexa)
	data, err := hex.DecodeString(hexa)
	checkErr(err, "func base64Encode param NULL")

	base := base64.StdEncoding.EncodeToString([]byte(data))
	log.Println("---base---:", base)

    return base
}

//base64 to hexa
func base64Decode(base string) (string) {
    data, err := base64.StdEncoding.DecodeString(base)
    checkErr(err, "func base64Decode param NULL")

    hexa := hex.EncodeToString(data)
    return hexa
}

//base64 to string
func base64DecodeStr(base string) (string) {
    bt, err := base64.StdEncoding.DecodeString(base)
    checkErr(err, "func base64Decode param NULL")

    str := string(bt)

    return str
}

//écriture d'un fichier
func writeDocument(dataDoc string, nameDoc string, pathDir string){

	docPath := filepath.Join(pathDir, nameDoc)
	f, err := os.Create(docPath)
	checkErr(err,"func writeDocument() using os.OpenFile()")

	defer f.Close()

	nb, err := f.WriteString(dataDoc)
	checkErr(err,"func writeDocument() using f.WriteString()")
	log.Println("---nombres de bytes écrits: ---", nb)
}

//lecture d'un fichier; param : path (chemin du fichier à lire); return: fi (le fichier)
func readDocument(path string) string{

	if _, err := os.Stat(path); err == nil {
		content, err := ioutil.ReadFile(path)
			checkErr(err, "func readDocument ioutil.ReadFile")

		return string(content)
	}else{
		return "no document!"
	}
}

func loadConfiguration(file string) (configStruct, error) {
    var conf configStruct
    configFile, err := os.Open(file)
    defer configFile.Close()
    if err != nil {
        return conf, err
    }
    jsonParser := json.NewDecoder(configFile)
    jsonParser.Decode(&conf)
    return conf, err
}

func checkIni() {

	_, err := loadConfiguration("config.json")
	checkErr(err, "impossible de lire le fichier de configuration")

	if _, err := os.Stat(Config.DirPath); os.IsNotExist(err) {
		// si le dossier n'existe pas, le créer //à placer au démarrage du programme

		folderPath := filepath.Join("C:/", "GEDdocuments/")
		err := os.MkdirAll(folderPath, os.ModePerm)
		checkErr(err, "func checkIni() using os.MkdirAll()")
		log.Println("----------nouveau dossier C:/GEDdocuments créé : ----")
	}
}

func main(){
	log.Println("----------Starting the service---")
	checkIni()

	var jwtMiddleware = jwtmiddleware.New(jwtmiddleware.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return mySigningKey, nil
		},
		SigningMethod: jwt.SigningMethodHS256,
	})

	router := mux.NewRouter()

	router.HandleFunc("/get-token/{userCode}", func(w http.ResponseWriter, r *http.Request){
		 GetTokenHandler(w, r)
	}).Methods("GET")
	router.HandleFunc("/interface", func(w http.ResponseWriter, r *http.Request){
		 formHandler(w, r)
	}).Methods("GET")

	router.Handle("/document/{guid}",jwtMiddleware.Handler(getDoc)).Methods("GET") //retourne un doc
	router.Handle("/document/upload",jwtMiddleware.Handler(uploadDoc)).Methods("POST") //post un doc

	router.Handle("/temp/{userCode}",jwtMiddleware.Handler(waitingDoc)).Methods("GET") //retourne les docs en attente de classement pour un personne
	router.Handle("/documentslist/{userCode}/{project}/{keywords}/{dateCreation}",jwtMiddleware.Handler(listDoc)).Methods("GET") //retourne une liste de docs selon un param
	router.Handle("/document/{guid}",jwtMiddleware.Handler(uploadDoc)).Methods("PUT") //modifie un doc
	router.Handle("/document/{guid}",jwtMiddleware.Handler(deleteDoc)).Methods("DELETE") //supprime un doc

	log.Fatal(http.ListenAndServe(":8080", router))
}

//userCode/{userCode}/project/{project}/keywords/{keywords}/dateCreation/{dateCreation}