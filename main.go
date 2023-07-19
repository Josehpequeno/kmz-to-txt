package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/tushar2708/altcsv"

	"encoding/xml"
)

type Stop struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

type SimpleData struct {
	Stops []Stop `xml:"SimpleData"`
}

type Placemark struct {
	Name string `xml:"name"`
	// Description string `xml:"description"`
	// LineString struct {
	// 	Coordinates string `xml:"coordinates"`
	// } `xml:"LineString"`
	StopsData []SimpleData `xml:"ExtendedData>SchemaData"`
}

type KML struct {
	XMLName            xml.Name    `xml:"kml"`
	Placemarks         []Placemark `xml:"Document>Placemark"`
	PlacemarksFolder   []Placemark `xml:"Document>Folder>Placemark"`
	PlacemarksDocument []Placemark `xml:"Folder>Document>Placemark"`
}

func main() {
	os.Chdir("./files")
	cmd := exec.Command("pwd")
	cmd.Stdin = strings.NewReader("")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	pwd := strings.Split(out.String(), "\n")[0]
	cmd = exec.Command("ls", "-ap")
	cmd.Stdin = strings.NewReader("")
	// var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	outArray := strings.Split(out.String(), "\n")
	for i := 0; i < len(outArray); i++ {
		arr := strings.Split(outArray[i], ".")
		if len(arr) < 2 {
			continue
		}
		if arr[1] != "kmz" {
			continue
		}
		kmzFile := pwd + "/" + outArray[i]

		err := extractKMLFromKMZ(kmzFile)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Conversion completed successfully!")
	}

	cmd = exec.Command("ls", "-ap")
	cmd.Stdin = strings.NewReader("")
	// var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	outArray = strings.Split(out.String(), "\n")

	//criar arquivos GTFS
	agencyFile, err := os.Create("agency.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer agencyFile.Close()

	routesFile, err := os.Create("routes.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer routesFile.Close()

	stopsFile, err := os.Create("stops.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer stopsFile.Close()

	schedulesFile, err := os.Create("schedules.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer schedulesFile.Close()

	//criar escritores CSV para os arquivos GTFS
	agencyWriter := altcsv.NewWriter(agencyFile)
	routesWriter := altcsv.NewWriter(routesFile)
	stopsWriter := altcsv.NewWriter(stopsFile)
	schedulesWriter := altcsv.NewWriter(schedulesFile)
	agencyWriter.AllQuotes = true
	routesWriter.AllQuotes = true
	stopsWriter.AllQuotes = true
	schedulesWriter.AllQuotes = true

	//cabeçalhos dos arquivos GTFS
	agencyWriter.Write([]string{"agency_id", "agency_name", "agency_url", "agency_timezone", "agency_lang"})
	agencyWriter.Write([]string{"0","example","https://example.br/", "America/Fortaleza", "pt"})
	agencyWriter.Flush()
	routesWriter.Write([]string{"route_id", "agency_id", "route_name"})
	stopsWriter.Write([]string{"stops_id", "stop_name", "stop_desc", "stop_lat", "stop_lon"})
	schedulesWriter.Write([]string{"route_id", "stop_id", "arrival_time"})

	for i := 0; i < len(outArray); i++ {
		arr := strings.Split(outArray[i], ".")
		if len(arr) < 2 {
			continue
		}
		if arr[1] != "kml" {
			continue
		}
		kmlFile := pwd + "/" + outArray[i]
		fmt.Println("file:", kmlFile, i)

		err := convertKMLToGTFS(kmlFile, routesWriter, stopsWriter, schedulesWriter)
		if err != nil {
			log.Fatal(err)
		}
	}

}

func extractKMLFromKMZ(path string) error {

	// abrir o arquivo kml
	kmzFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer kmzFile.Close() // O defer é seguido por uma chamada de função que será adiada para execução. A função será avaliada imediatamente, mas sua execução será adiada.

	fi, err := kmzFile.Stat()
	if err != nil {
		return err
	}

	// cria um leitor de arquivos zip
	zipReader, err := zip.NewReader(kmzFile, fi.Size())
	if err != nil {
		return err
	}

	// procurar o arquivo KML dentro do KMZ
	var kmlData []byte
	for _, file := range zipReader.File {
		if strings.HasSuffix(file.Name, ".kml") {
			// Abrir o arquivo KML
			path, err := file.Open()
			if err != nil {
				return err
			}
			defer path.Close()

			kmlData, err = io.ReadAll(path)
			if err != nil {
				return err
			}

			break
		}
	}
	// Salver o arquivo kml extraído
	err = os.WriteFile(strings.Split(path, ".")[0]+".kml", kmlData, 0644)
	if err != nil {
		return err
	}

	return nil
}

func convertKMLToGTFS(kmlFile string, routesWriter *altcsv.Writer, stopsWriter *altcsv.Writer, schedulesWriter *altcsv.Writer) error {
	// Abrir o arquivo KML
	xmlFile, err := os.Open(kmlFile)
	if err != nil {
		return err
	}
	defer xmlFile.Close()

	// Ler o conteúdo do KML
	// k := kml.KML(kmlData)
	// kml.Data(kmlData)

	xmlData, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		return err
	}

	//processar as features do kml
	var kml KML
	err = xml.Unmarshal(xmlData, &kml)
	if err != nil {
		return err
	}

	var array []Placemark
	array = append(array, kml.Placemarks...)
	array = append(array, kml.PlacemarksFolder...)
	array = append(array, kml.PlacemarksDocument...)
	

	for _, placemark := range array {
		routeID := placemark.Name
		routeName := placemark.Name
		if len(placemark.StopsData) > 0 {
			for _, stopData := range placemark.StopsData {
				stopID := ""
				stopName := ""
				stopDesc := ""
				stopLat := ""
				stopLon := ""
				for _, stop := range stopData.Stops {
					if stop.Name == "stop_id" {
						stopID = stop.Value
					}
					if stop.Name == "stop_name" {
						stopName = stop.Value
					}
					if stop.Name == "stop_desc" {
						stopDesc = stop.Value
					}
					if stop.Name == "stop_lat" {
						stopLat = stop.Value
					}
					if stop.Name == "stop_lon" {
						stopLon = stop.Value
					}
				}
				stopsWriter.Write([]string{stopID, stopName, stopDesc, stopLat, stopLon})
			}
		}

		routesWriter.Write([]string{routeID, "1", routeName})
	
	}

	// finalizar escrita dos arquivos GTFS
	routesWriter.Flush()
	stopsWriter.Flush()
	schedulesWriter.Flush()

	if err := routesWriter.Error(); err != nil {
		return err
	}
	if err := stopsWriter.Error(); err != nil {
		return err
	}
	if err := schedulesWriter.Error(); err != nil {
		return err
	}

	return nil

}
