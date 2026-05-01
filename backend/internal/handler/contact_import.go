package handler

import (
	"encoding/csv"
	"errors"
	"io"
	"strings"

	"backend/internal/dto"
	"backend/internal/repo"
)

type ParsedImport struct {
	Contacts []repo.ImportContact
	Errors   []dto.ImportError
	Total    int
}

func ParseContactCSV(data string) (*ParsedImport, error) {
	reader := csv.NewReader(strings.NewReader(data))
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		return nil, err
	}

	nameIdx := -1
	emailIdx := -1
	phoneIdx := -1
	for i, col := range header {
		switch strings.ToLower(strings.TrimSpace(col)) {
		case "name", "nome":
			nameIdx = i
		case "email", "e-mail":
			emailIdx = i
		case "phone", "phone_number", "telefone":
			phoneIdx = i
		}
	}
	if nameIdx == -1 && emailIdx == -1 {
		return nil, ErrMissingRequiredColumn
	}

	var result ParsedImport
	rowNum := 1

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			rowNum++
			result.Errors = append(result.Errors, dto.ImportError{Row: rowNum, Reason: "malformed row"})
			continue
		}
		rowNum++

		name := ""
		email := ""
		phone := ""
		if nameIdx >= 0 && nameIdx < len(record) {
			name = strings.TrimSpace(record[nameIdx])
		}
		if emailIdx >= 0 && emailIdx < len(record) {
			email = strings.TrimSpace(record[emailIdx])
		}
		if phoneIdx >= 0 && phoneIdx < len(record) {
			phone = strings.TrimSpace(record[phoneIdx])
		}

		if name == "" && email == "" {
			result.Errors = append(result.Errors, dto.ImportError{Row: rowNum, Reason: "name and email are empty"})
			continue
		}

		result.Contacts = append(result.Contacts, repo.ImportContact{Name: name, Email: email, Phone: phone})
		result.Total++
	}

	return &result, nil
}

var ErrMissingRequiredColumn = errors.New("missing required column (name or email)")
