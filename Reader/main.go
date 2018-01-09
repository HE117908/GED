/*
Created by: Alexandre Vanham
Modified by: Alexandre Vanham
Date: 15-11-2017
Version: 0.9
*/
package main

import (
	//"fmt"
	"log"
	"os"
	"net/http"
	"strconv"
	"io/ioutil"
	"path/filepath"
	"encoding/json"
	"bytes"
	"encoding/base64"
)

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

var path = "test\\"

func readDirDocs(){
	var myDoc iDoc
	fi := getFiles(path)

	for _, fi := range fi {
	    if fi.Mode().IsRegular() {
			//fmt.Println("---fichier sys : ---%+v\n", fi.Sys())
			myDoc.Id_Code = "0"
			myDoc.Idt_Code = "MANUAL"
			myDoc.Id_Comment = ""
			myDoc.Id_Path = path + fi.Name()
			myDoc.Id_FileName = fi.Name()
			myDoc.Id_Classed = "0"
			myDoc.Id_Creation_Date = fi.ModTime().String()
			myDoc.Id_Version = "0"
			myDoc.Id_Size = strconv.Itoa(int(fi.Size()))
			myDoc.Id_JSon = ""
			contentDoc := readDocument(path + fi.Name())
			myDoc.Id_Binary = base64Encode(string(contentDoc))
			ext := filepath.Ext(fi.Name())
			myDoc.Id_BinaryType = ext[1:]
			myDoc.Id_BinaryLg = "FR"

			//create json
			//json.NewEncoder(jsonDoc).Encode(myDoc)
			jsonDoc, _ := json.Marshal(myDoc)

	        if(fi.IsDir() == false){
	        	url := "http://localhost:8080/document/upload"
				postDoc(fi, url, getToken(), string(jsonDoc))
	        }else{
	        	log.Println("---not a file, it is a directory---")
	        }
	    }
	}

}

//renvoye le token du serveur WS; return token signé
func getToken()string{
	url := "http://localhost:8080/get-token/AV"

	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
		checkErr(err)
	resp, err := client.Do(req)
		checkErr(err)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
		checkErr(err)
	token := string(body)

	if(token != ""){
		log.Println("---token received : ----"/* + token*/)
	}else{
		log.Println("---!no token received! : ----"/* + token*/)
	}

    return token
}

//utilisation service web; requête enregistrement doc; params : fi (le fichier), url (l'url utilisé pour le post), token (le token signé de sécurité); return /
func postDoc(fi os.FileInfo, url string, token string, jsonDoc string) {


	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(jsonDoc)))
		checkErr(err)
	req.Header.Add("Authorization", "Bearer " + token)

	client := &http.Client{}
	resp, err := client.Do(req)
		checkErr(err)

	log.Println("---func postDoc() --- : document added : " + fi.Name())// + "url" + url + "token" + token + "json" + jsonDoc)

	defer resp.Body.Close()

	return
}

//lecture d'un directory; param : path (directory à lire); return: fi (un slice des fichiers du dir)
func getFiles(path string) []os.FileInfo{
	d, err := os.Open(path)
		checkErr(err)
	defer d.Close()
	fi, err := d.Readdir(-1)
		checkErr(err)

	return fi
}

//lecture d'un fichier; param : path (chemin du fichier à lire); return: fi (le fichier)
func readDocument(path string) string{

	if _, err := os.Stat(path); err == nil {
		content, err := ioutil.ReadFile(path)
			checkErr(err)

		return string(content)
	}else{
		return "no document!"
	}
}

//str to base64
func base64Encode(st string) string {

	base := base64.StdEncoding.EncodeToString([]byte(st))
	//log.Println("---base---:", base)

    return base
}

//func de gestion d'erreur; params: error, (msg)
func checkErr(err error) {
	if err != nil {
		log.Println("Error: ", err)
	}
}

func main() {
	//token := getToken()

	readDirDocs()
}
