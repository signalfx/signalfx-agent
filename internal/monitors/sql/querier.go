package sql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/sirupsen/logrus"
)

type querier struct {
	query                     *Query
	valueColumnNamesToMetrics map[string]*Metric
	metricToIndex             map[*Metric]int
	dimensionColumnSets       []map[string]bool
	datapoints                []*datapoint.Datapoint
	dimensions                [][]*types.Dimension
	rowSliceCached            []interface{}
	logger                    logrus.FieldLogger
	logQueries                bool
}

func newQuerier(query *Query, logQueries bool) *querier {
	valueColumnNamesToMetrics := map[string]*Metric{}
	metricToIndex := map[*Metric]int{}

	for i, m := range query.Metrics {
		valueColumnNamesToMetrics[strings.ToLower(m.ValueColumn)] = &query.Metrics[i]
		metricToIndex[&query.Metrics[i]] = i
	}

	dimensionColumnSets := make([]map[string]bool, len(query.Metrics))
	for i := range dimensionColumnSets {
		dimensionColumnSets[i] = map[string]bool{}
	}

	dimensions := make([][]*types.Dimension, len(query.Metrics))

	// Make a set of cloneable datapoints that already have metric name and
	// type set since it never changes with the same metric config.
	dps := make([]*datapoint.Datapoint, len(query.Metrics))
	for i, m := range query.Metrics {
		typ := datapoint.Gauge
		if m.IsCumulative {
			typ = datapoint.Counter
		}
		dps[i] = datapoint.New(m.MetricName, nil, nil, typ, time.Time{})

		for _, dim := range m.DimensionColumns {
			dimensionColumnSets[i][strings.ToLower(dim)] = true
		}

		dimensions[i] = make([]*types.Dimension, 0)
		for dim, propColumns := range m.DimensionPropertyColumns {
			dimColumn := strings.ToLower(dim)
			if dimensionColumnSets[i][dimColumn] {
				props := make(map[string]string)
				for _, p := range propColumns {
					props[strings.ToLower(p)] = ""
				}
				dimensions[i] = append(dimensions[i], &types.Dimension{
					Name:       dimColumn,
					Properties: props,
				})
			}
			// TODO: Should we handle the case when the given dimension column is not really used as a dimension somehow here?
		}
	}

	return &querier{
		query: query,
		// Preallocate the slice and reuse it since it will only be used
		// serially.
		datapoints:                dps,
		valueColumnNamesToMetrics: valueColumnNamesToMetrics,
		metricToIndex:             metricToIndex,
		dimensionColumnSets:       dimensionColumnSets,
		dimensions:                dimensions,
		logger:                    logger.WithField("statement", query.Query),
		logQueries:                logQueries,
	}
}

func (q *querier) doQuery(ctx context.Context, database *sql.DB, output types.Output) error {
	rows, err := database.QueryContext(ctx, q.query.Query, q.query.Params...)
	if err != nil {
		return fmt.Errorf("error executing statement %s: %v", q.query.Query, err)
	}
	for rows.Next() {
		dps, dims, err := q.convertCurrentRowToDatapointAndDimensions(rows)
		if err != nil {
			return err
		}

		// Make a new slice since this montior reuses the backing array of dps
		outDPs := make([]*datapoint.Datapoint, 0, len(dps))
		for i := range dps {
			if dps[i].Value == nil {
				q.logger.Warnf("Metric %s's value column '%s' did not correspond to a value",
					q.query.Metrics[i].MetricName, q.query.Metrics[i].ValueColumn)
				continue
			}
			outDPs = append(outDPs, dps[i])
		}
		output.SendDatapoints(outDPs...)

		for i := range dims {
			for _, dim := range dims[i] {
				output.SendDimensionUpdate(dim)
			}
		}
	}
	return rows.Close()
}

func (q *querier) convertCurrentRowToDatapointAndDimensions(rows *sql.Rows) ([]*datapoint.Datapoint, [][]*types.Dimension, error) {
	rowScanSlice, err := q.getRowSlice(rows)
	if err != nil {
		return nil, nil, err
	}

	columnNames, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}

	if err := rows.Scan(rowScanSlice...); err != nil {
		return nil, nil, err
	}
	if q.logQueries {
		q.logger.Info("Got results %s", spew.Sdump(rowScanSlice))
	}

	// Clone all datapoints before updating them
	for i := range q.datapoints {
		dpCopy := *q.datapoints[i]
		q.datapoints[i] = &dpCopy
		q.datapoints[i].Dimensions = map[string]string{}
		q.datapoints[i].Meta = map[interface{}]interface{}{}
	}

	// Clone all properties before updating them
	for i := range q.dimensions {
		for j := range q.dimensions[i] {
			props := make(map[string]string)
			for propName := range q.dimensions[i][j].Properties {
				props[propName] = ""
			}
			q.dimensions[i][j] = &types.Dimension{
				Name:       q.dimensions[i][j].Name,
				Properties: props,
			}
		}
	}

	for i := range rowScanSlice {
		switch v := rowScanSlice[i].(type) {
		case *sql.NullFloat64:
			if !v.Valid {
				return nil, nil, fmt.Errorf("column %d is null", i)
			}

			metric, ok := q.valueColumnNamesToMetrics[strings.ToLower(columnNames[i])]
			if !ok || metric == nil {
				// This is a logical error in the code, not user input error
				panic("valueColumn was not properly mapped to metric")
			}

			dp := q.datapoints[q.metricToIndex[metric]]
			dp.Value = datapoint.NewFloatValue(v.Float64)

		case *sql.NullString:
			dimVal := v.String
			if !v.Valid {
				// Make sure the value gets properly blanked out since we are
				// reusing rowScanSlice between rows/queries.
				dimVal = ""
			}
			for j := range q.query.Metrics {
				for _, dim := range q.dimensions[j] {
					if strings.EqualFold(dim.Name, columnNames[i]) {
						dim.Name = columnNames[i]
						dim.Value = dimVal
					}
					for k := range dim.Properties {
						if strings.EqualFold(columnNames[i], k) {
							dim.Properties[k] = utils.TruncateDimensionValue(dimVal)
						}
					}
				}
				if !q.dimensionColumnSets[j][strings.ToLower(columnNames[i])] {
					continue
				}

				q.datapoints[j].Dimensions[columnNames[i]] = dimVal
			}
		}
	}

	return q.datapoints, q.dimensions, nil
}

func (q *querier) getRowSlice(rows *sql.Rows) ([]interface{}, error) {
	if q.rowSliceCached != nil {
		return q.rowSliceCached, nil
	}

	cts, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	dimColsSeen := map[string]bool{}
	propColsSeen := map[string]bool{}
	rowSlice := make([]interface{}, len(cts))
OUTER:
	for i, ct := range cts {
		for _, metric := range q.query.Metrics {

			if strings.EqualFold(ct.Name(), metric.ValueColumn) {
				// Values are always numeric
				rowSlice[i] = &sql.NullFloat64{}
				// Can't also be a dimension column or value in another metric
				continue OUTER
			}

			for _, propertyColumns := range metric.DimensionPropertyColumns {
				for _, colName := range propertyColumns {
					if strings.EqualFold(ct.Name(), colName) {
						propColsSeen[colName] = true
					}
				}
			}

			for _, colName := range metric.DimensionColumns {
				if strings.EqualFold(ct.Name(), colName) {
					dimColsSeen[colName] = true
					rowSlice[i] = &sql.NullString{}
					// Cannot also be a value column if dimension
					continue OUTER
				}
			}

		}
		// This column is unused in generating metrics so just make it a string
		rowSlice[i] = &sql.NullString{}
	}

	for _, metric := range q.query.Metrics {
		for _, dimCol := range metric.DimensionColumns {
			if !dimColsSeen[dimCol] {
				return nil, fmt.Errorf("dimension column '%s' does not exist", dimCol)
			}
		}

		for _, propertyColumns := range metric.DimensionPropertyColumns {
			for _, propCol := range propertyColumns {
				if !propColsSeen[propCol] {
					return nil, fmt.Errorf("property column '%s' does not exist", propCol)
				}
			}
		}
	}

	q.rowSliceCached = rowSlice
	return rowSlice, nil
}
