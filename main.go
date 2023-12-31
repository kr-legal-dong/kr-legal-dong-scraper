package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"
	"kr-legal-dong-scraper/model"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/shakinm/xlsReader/xls"
)

func main() {
	client := http.Client{
		Timeout: 2 * time.Minute,
	}

	url := "https://www.code.go.kr/stdcode/regCodeFileDown.do?cPage=1&pageSize=100000&chkHigh=0&chkLow=0&disuseAt=ALL"
	var zipBytes []byte

	zipBytes, err := retry.DoWithData(
		func() ([]byte, error) {
			zipFile, err := client.Get(url)
			if err != nil {
				return nil, err
			}

			defer zipFile.Body.Close()

			zipBytes, err = io.ReadAll(zipFile.Body)
			if err != nil {
				return nil, err
			}

			return zipBytes, nil
		},
	)

	if err != nil {
		panic(err)
	}

	reader := bytes.NewReader(zipBytes)

	zipReader, err := zip.NewReader(reader, int64(len(zipBytes)))
	if err != nil {
		panic(err)
	}

	if len(zipReader.File) == 0 {
		panic(err)
	}

	xlsFile, err := zipReader.File[0].Open()
	if err != nil {
		panic(err)
	}

	defer xlsFile.Close()

	xlsBytes, err := io.ReadAll(xlsFile)
	if err != nil {
		panic(err)
	}

	os.WriteFile("tmp/tmp.xls", xlsBytes, 0644)

	workbook, err := xls.OpenFile("tmp/tmp.xls")
	if err != nil {
		panic(err)
	}

	sheet, err := workbook.GetSheet(0)
	if err != nil {
		panic(err)
	}

	si := []model.Si{}

	rows := sheet.GetRows()

	for _, row := range rows {
		cells := row.GetCols()

		if cells[3] != nil && cells[3].GetString() == "0000000000" {
			si = append(si, model.Si{
				Code:   cells[0].GetString(),
				Name:   cells[1].GetString(),
				Active: cells[2].GetString() == "현존",
			})
		}
	}

	gu := []model.Gu{}

	for _, row := range rows {
		cells := row.GetCols()

		if cells[3] != nil {

			for _, si := range si {
				if si.Code == cells[3].GetString() {
					name := cells[1].GetString()
					name = strings.ReplaceAll(name, si.Name, "")
					name = strings.TrimSpace(name)

					gu = append(gu, model.Gu{
						Code:     cells[0].GetString(),
						SiCode:   si.Code,
						SiName:   si.Name,
						FullName: cells[1].GetString(),
						Name:     name,
						Active:   cells[2].GetString() == "현존",
					})
				}
			}
		}
	}

	dong := []model.Dong{}

	for _, row := range rows {
		cells := row.GetCols()

		if cells[2] != nil {

			for _, gu := range gu {
				if gu.Code == cells[3].GetString() {
					name := cells[1].GetString()
					name = strings.ReplaceAll(name, gu.FullName, "")
					name = strings.TrimSpace(name)

					dong = append(dong, model.Dong{
						Code:     cells[0].GetString(),
						SiCode:   gu.SiCode,
						SiName:   gu.SiName,
						GuCode:   gu.Code,
						GuName:   gu.Name,
						FullName: cells[1].GetString(),
						Name:     name,
						Active:   cells[2].GetString() == "현존",
					})
				}
			}
		}
	}

	detail := []model.Detail{}

	for _, row := range rows {
		cells := row.GetCols()

		if cells[2] != nil {

			for _, dong := range dong {
				if dong.Code == cells[3].GetString() {
					name := cells[1].GetString()
					name = strings.ReplaceAll(name, dong.FullName, "")
					name = strings.TrimSpace(name)

					detail = append(detail, model.Detail{
						Code:     cells[0].GetString(),
						SiCode:   dong.SiCode,
						SiName:   dong.SiName,
						GuCode:   dong.GuCode,
						GuName:   dong.GuName,
						DongCode: dong.Code,
						DongName: dong.Name,
						FullName: cells[1].GetString(),
						Name:     name,
						Active:   cells[2].GetString() == "현존",
					})
				}
			}
		}
	}

	siJson, _ := json.MarshalIndent(si, "", " ")
	err = os.WriteFile("tmp/si.json", siJson, 0644)
	if err != nil {
		panic(err)
	}

	guJson, _ := json.MarshalIndent(gu, "", " ")
	err = os.WriteFile("tmp/gu.json", guJson, 0644)
	if err != nil {
		panic(err)
	}

	dongJson, _ := json.MarshalIndent(dong, "", " ")
	err = os.WriteFile("tmp/dong.json", dongJson, 0644)
	if err != nil {
		panic(err)
	}

	detailJson, _ := json.MarshalIndent(detail, "", " ")
	err = os.WriteFile("tmp/detail.json", detailJson, 0644)
	if err != nil {
		panic(err)
	}

	siCsv, err := os.Create("tmp/si.csv")
	if err != nil {
		panic(err)
	}

	siCsvWriter := csv.NewWriter(bufio.NewWriter(siCsv))
	defer siCsvWriter.Flush()

	siCsvWriter.Write([]string{"code", "name", "active"})
	for _, data := range si {
		siCsvWriter.Write([]string{data.Code, data.Name, strconv.FormatBool(data.Active)})
	}

	guCsv, err := os.Create("tmp/gu.csv")
	if err != nil {
		panic(err)
	}

	guCsvWriter := csv.NewWriter(bufio.NewWriter(guCsv))
	defer guCsvWriter.Flush()

	guCsvWriter.Write([]string{"code", "siCode", "siName", "fullName", "name", "active"})
	for _, data := range gu {
		guCsvWriter.Write([]string{data.Code, data.SiCode, data.SiName, data.FullName, data.Name, strconv.FormatBool(data.Active)})
	}

	dongCsv, err := os.Create("tmp/dong.csv")
	if err != nil {
		panic(err)
	}

	dongCsvWriter := csv.NewWriter(bufio.NewWriter(dongCsv))
	defer dongCsvWriter.Flush()

	dongCsvWriter.Write([]string{"code", "siCode", "siName", "guCode", "guName", "fullName", "name", "active"})
	for _, data := range dong {
		dongCsvWriter.Write([]string{data.Code, data.SiCode, data.SiName, data.GuCode, data.GuName, data.FullName, data.Name, strconv.FormatBool(data.Active)})
	}

	detailCsv, err := os.Create("tmp/detail.csv")
	if err != nil {
		panic(err)
	}

	detailCsvWriter := csv.NewWriter(bufio.NewWriter(detailCsv))
	defer detailCsvWriter.Flush()

	detailCsvWriter.Write([]string{"code", "siCode", "siName", "guCode", "guName", "dongCode", "dongName", "fullName", "name", "active"})
	for _, data := range detail {
		detailCsvWriter.Write([]string{data.Code, data.SiCode, data.SiName, data.GuCode, data.GuName, data.DongCode, data.DongName, data.FullName, data.Name, strconv.FormatBool(data.Active)})
	}
}
