package repository

import (
	"events-app/data/models"
	"fmt"
	"strconv"
	"strings"
)

// buildQuery constructs a formatted and parameterized sql string from the
// given query parameters. It returns the finished sql string, and the values to be
// passed alongside the query. It returns an error if any of the query
// parameters fail to validate against the model's jsonMap.
func buildQueryClauses(queryParams map[string]string, m models.Model) (clauses string, sqlVals []interface{}, err error) {
	placeholderIndex := 1
	jsonMap := models.MapJsonTagsToDB(m)
	// Filtering
	whereClause, sqlVals, placeholderIndex, err := buildWhereClause(queryParams, placeholderIndex, jsonMap)
	if err != nil {
		return "", nil, err
	}

	// Sorting
	sort, order, err := buildSortingClause(queryParams, jsonMap)
	if err != nil {
		return "", nil, err
	}
	orderClause := fmt.Sprintf("ORDER BY %s %s", sort, order)

	// Pagination
	limit, offset, err := buildPaginationClause(queryParams)
	if err != nil {
		return "", nil, err
	}
	paginationClause := fmt.Sprintf("LIMIT $%d OFFSET $%d", placeholderIndex, placeholderIndex+1)
	sqlVals = append(sqlVals, limit, offset)

	if whereClause != "" {
		clauses = fmt.Sprintf("%s %s %s", whereClause, orderClause, paginationClause)
	} else {
		clauses = fmt.Sprintf("%s %s", orderClause, paginationClause)
	}

	return clauses, sqlVals, nil
}

// buildWhereClause constructs a formatted and parameterized sql WHERE clause.
// It returns the finished WHERE clause, the values to be ultimately passed
// alongside the query, and the current placeholder count. If there are no
// search conditions in the query parameters, it returns an empty string for the
// WHERE clause.
func buildWhereClause(queryParams map[string]string, phIndex int, jsonMap map[string]string) (whereClause string, sqlVals []interface{}, placeholderIndex int, err error) {
	whereClauseParts := []string{}

	for key, value := range queryParams {
		// Skip these for later handling
		if key == "sortBy" || key == "limit" || key == "offset" {
			continue
		}

		// Parse the operator and db column name from the key
		operator, dbColumn, value, err := parseOperatorAndKey(key, value, jsonMap)
		if err != nil {
			return "", nil, 0, err
		}
		// We need to handle the IN operator differently because its list of
		// values is of variable length (e.g. name_anyOf=Tom,Dick,Harry;
		// name_anyOf=Tom,Dick)
		if operator == "IN" {
			whereClauseParts, sqlVals, phIndex, err = handleInOperator(key, value, phIndex, whereClauseParts, sqlVals, jsonMap)
			if err != nil {
				return "", nil, 0, err
			}
			// Skip the rest of the loop because we've already handled the IN operator
			continue
		}

		// assemble the clause-part
		whereClauseParts = append(whereClauseParts, fmt.Sprintf("%s %s $%d", dbColumn, operator, phIndex))
		// Perform type conversion on numerical characters before appending to vals slice
		formattedVal := convertValueIfNumeric(value)
		sqlVals = append(sqlVals, formattedVal)
		phIndex++
	}

	whereClause = ""
	if len(whereClauseParts) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauseParts, " AND ")
	}

	return whereClause, sqlVals, phIndex, nil
}

// parseOperatorAndKey determines the SQL operator and strips the operator
// suffix from the key. It returns the operator, the key's database column
// mapping, and the modified value (if applicable). It returns an error if the
// key does not exist in the model's jsonMap.
func parseOperatorAndKey(key, value string, jsonMap map[string]string) (operator, dbColumn string, modifiedValue string, err error) {
	operator = "="
	modifiedValue = value

	if strings.HasSuffix(key, "_ne") {
		operator = "!="
		key = strings.TrimSuffix(key, "_ne")

	} else if strings.HasSuffix(key, "_lt") {
		operator = "<"
		key = strings.TrimSuffix(key, "_lt")

	} else if strings.HasSuffix(key, "_gt") {
		operator = ">"
		key = strings.TrimSuffix(key, "_gt")

	} else if strings.HasSuffix(key, "_lte") {
		operator = "<="
		key = strings.TrimSuffix(key, "_lte")

	} else if strings.HasSuffix(key, "_gte") {
		operator = ">="
		key = strings.TrimSuffix(key, "_gte")

	} else if strings.HasSuffix(key, "_contains") {
		operator = "LIKE"
		key = strings.TrimSuffix(key, "_contains")
		modifiedValue = "%" + value + "%"

	} else if strings.HasSuffix(key, "_anyOf") {
		operator = "IN"
		key = strings.TrimSuffix(key, "_anyOf")
	}

	if err := validateQueryParam(key, jsonMap); err != nil {
		return "", "", "", err
	}

	// Map the JSON tag to the DB column name and return that for the query
	dbColumn = jsonMap[key]

	return operator, dbColumn, modifiedValue, nil
}

// handleInOperator builds a WHERE clause part, from a list of comma-separated
// values, for the IN operator  It is a helper for buildWhereClause. It returns
// the still-under-construction WHERE clause parts, the values to be ultimately passed
// alongside the query, and the current placeholder count.
func handleInOperator(key, value string, phIndex int, whereClauseParts []string, sqlVals []interface{}, jsonMap map[string]string) ([]string, []interface{}, int, error) {
	anyOfValuesList := strings.Split(value, ",")
	placeholders := []string{}

	for _, v := range anyOfValuesList {
		placeholders = append(placeholders, fmt.Sprintf("$%d", phIndex))
		// Perform numerical type conversion here if needed
		formattedVal := convertValueIfNumeric(v)
		sqlVals = append(sqlVals, formattedVal)
		phIndex++
	}

	key = strings.TrimSuffix(key, "_anyOf")
	if err := validateQueryParam(key, jsonMap); err != nil {
		return nil, nil, 0, err
	}

	dbColumn := jsonMap[key]
	whereClauseParts = append(whereClauseParts, fmt.Sprintf("%s IN (%s)", dbColumn, strings.Join(placeholders, ",")))
	return whereClauseParts, sqlVals, phIndex, nil
}

func buildSortingClause(queryParams map[string]string, jsonMap map[string]string) (string, string, error) {
	sort := queryParams["sortBy"]
	order := "ASC"
	if strings.HasPrefix(sort, "-") {
		order = "DESC"
		sort = strings.TrimPrefix(sort, "-")
	}
	if sort == "" {
		sort = "id"
	}

	if err := validateQueryParam(sort, jsonMap); err != nil {
		return "", "", fmt.Errorf("invalid sort value: %v", sort)
	}

	sort = jsonMap[sort]
	return sort, order, nil
}

func buildPaginationClause(queryParams map[string]string) (int, int, error) {
	limit := 10
	offset := 0
	if l, ok := queryParams["limit"]; ok {
		var err error
		limit, err = strconv.Atoi(l)
		if err != nil {
			return 0, 0, fmt.Errorf("pagination err; limit must be a number: %v", err)
		}
	}
	if o, ok := queryParams["offset"]; ok {
		var err error
		offset, err = strconv.Atoi(o)
		if err != nil {
			return 0, 0, fmt.Errorf("pagination err; offset must be a number: %v", err)
		}
	}
	return limit, offset, nil
}

func convertValueIfNumeric(value string) interface{} {
	if intValue, err := strconv.Atoi(value); err == nil {
		return intValue
	} else if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
		return floatValue
	}
	return value
}

func validateQueryParam(key string, jsonMap map[string]string) error {
	if jsonMap[key] == "" {
		return fmt.Errorf("invalid query parameter: %s", key)
	}
	return nil
}
