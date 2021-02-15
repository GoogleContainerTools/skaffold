/*
Copyright 2021 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	dashboard "cloud.google.com/go/monitoring/dashboard/apiv1"
	"google.golang.org/api/iterator"
	dashboardpb "google.golang.org/genproto/googleapis/monitoring/dashboard/v1"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
)

const (
	dashboardDisplayName = "Skaffold Flags"
)

var projectID string

func main() {
	ctx := context.Background()
	dashboardClient, err := dashboard.NewDashboardsClient(ctx)

	if len(os.Args) < 2 {
		projectID = os.Getenv("GCP_PROJECT_ID")
		if projectID == "" {
			panic(fmt.Errorf("did not specify the project as an argument"))
		}
	} else {
		projectID = os.Args[1]
	}

	parent := fmt.Sprintf("projects/%s", projectID)
	listDBReq := dashboardpb.ListDashboardsRequest{Parent: parent}
	dbIterator := dashboardClient.ListDashboards(ctx, &listDBReq)
	if err != nil {
		panic(err)
	}

	var flagsDB *dashboardpb.Dashboard
	for db, err := dbIterator.Next(); err != iterator.Done; db, err = dbIterator.Next() {
		if err != nil || db == nil {
			panic(err)
		}
		if db.DisplayName == dashboardDisplayName {
			flagsDB = db
			break
		}
	}

	db := createFlagDashboard()

	if flagsDB == nil {
		createDBReq := dashboardpb.CreateDashboardRequest{Dashboard: db, Parent: parent}
		_, err = dashboardClient.CreateDashboard(ctx, &createDBReq)
	} else {
		db.Name = flagsDB.Name
		db.Etag = flagsDB.Etag
		patchDBReq := dashboardpb.UpdateDashboardRequest{Dashboard: db}
		_, err = dashboardClient.UpdateDashboard(ctx, &patchDBReq)
	}
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
}

func createFlagDashboard() *dashboardpb.Dashboard {
	var widgets []*dashboardpb.Widget
	for command := range instrumentation.MeteredCommands {
		widget := createWidgetForCommand(command)
		if widget != nil {
			widgets = append(widgets, widget)
		}
	}

	return &dashboardpb.Dashboard{
		DisplayName: dashboardDisplayName,
		Layout: &dashboardpb.Dashboard_GridLayout{
			GridLayout: &dashboardpb.GridLayout{
				Columns: 2,
				Widgets: widgets,
			},
		},
	}
}

func createWidgetForCommand(command string) *dashboardpb.Widget {
	var dataSets []*dashboardpb.XyChart_DataSet

	flagMetricName := "flags"
	dataSets = append(dataSets,
		createDataSetWithTimeSeriesFilter(
			createTimeSeriesFilter(
				flagMetricName,
				fmt.Sprintf("metric.label.\"command\"=\"%s\"", command),
				"metric.label.\"flag_value\"",
				"metric.label.\"flag_name\"")))

	if len(dataSets) == 0 {
		return nil
	}

	return createXYWidget("Total flags for "+command, dataSets)
}

func createXYWidget(widgetTitle string, dataSets []*dashboardpb.XyChart_DataSet) *dashboardpb.Widget {
	return &dashboardpb.Widget{
		Title: widgetTitle,
		Content: &dashboardpb.Widget_XyChart{
			XyChart: &dashboardpb.XyChart{
				YAxis: &dashboardpb.XyChart_Axis{
					Label: "Appearances",
					Scale: dashboardpb.XyChart_Axis_LINEAR,
				},
				ChartOptions: &dashboardpb.ChartOptions{Mode: dashboardpb.ChartOptions_COLOR},
				DataSets:     dataSets,
			},
		},
	}
}

func createDataSetWithTimeSeriesFilter(filter *dashboardpb.TimeSeriesFilter) *dashboardpb.XyChart_DataSet {
	return &dashboardpb.XyChart_DataSet{
		TimeSeriesQuery: &dashboardpb.TimeSeriesQuery{
			Source: &dashboardpb.TimeSeriesQuery_TimeSeriesFilter{TimeSeriesFilter: filter},
		},
		PlotType:           dashboardpb.XyChart_DataSet_LINE,
		MinAlignmentPeriod: durationpb.New(time.Second * 60),
	}
}

func createTimeSeriesFilter(flagMetricName string, labelFilter string, groupBy ...string) *dashboardpb.TimeSeriesFilter {
	return &dashboardpb.TimeSeriesFilter{
		Filter: fmt.Sprintf("metric.type=\"custom.googleapis.com/skaffold/%s\" resource.type=\"global\" %s", flagMetricName, labelFilter),
		Aggregation: &dashboardpb.Aggregation{
			PerSeriesAligner:   dashboardpb.Aggregation_ALIGN_MEAN,
			CrossSeriesReducer: dashboardpb.Aggregation_REDUCE_MEAN,
			GroupByFields:      groupBy,
		},
	}
}
