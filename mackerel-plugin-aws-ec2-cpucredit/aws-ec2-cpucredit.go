package main

import (
	"errors"
	"flag"
	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/cloudwatch"
	mp "github.com/mackerelio/go-mackerel-plugin"
	"os"
	"time"
)

var graphdef map[string](mp.Graphs) = map[string](mp.Graphs){
	"ec2.cpucredit": mp.Graphs{
		Label: "EC2 CPU Credit",
		Unit:  "float",
		Metrics: [](mp.Metrics){
			mp.Metrics{Name: "usage", Label: "Usage", Diff: false},
			mp.Metrics{Name: "balance", Label: "Balance", Diff: false},
		},
	},
}

type CPUCreditPlugin struct {
	Region          string
	AccessKeyId     string
	SecretAccessKey string
	InstanceId      string
}

func GetLastPointAverage(cw *cloudwatch.CloudWatch, dimension *cloudwatch.Dimension, metricName string) (float64, error) {
	namespace := "AWS/EC2"
	now := time.Now()
	prev := now.Add(time.Duration(600) * time.Second * -1) // 10 min (to fetch at least 1 data-point)

	request := &cloudwatch.GetMetricStatisticsRequest{
		Dimensions: []cloudwatch.Dimension{*dimension},
		EndTime:    now,
		StartTime:  prev,
		MetricName: metricName,
		Period:     60,
		Statistics: []string{"Average"},
		Namespace:  namespace,
	}

	response, err := cw.GetMetricStatistics(request)
	if err != nil {
		return 0, err
	}

	datapoints := response.GetMetricStatisticsResult.Datapoints
	if len(datapoints) == 0 {
		return 0, errors.New("fetched no datapoints")
	}

	latest := time.Unix(0, 0)
	var latestVal float64
	for _, dp := range datapoints {
		if dp.Timestamp.Before(latest) {
			continue
		}

		latest = dp.Timestamp
		latestVal = dp.Average
	}

	return latestVal, nil
}

func (p CPUCreditPlugin) FetchMetrics() (map[string]float64, error) {
	region := aws.Regions[p.Region]
	dimension := &cloudwatch.Dimension{
		Name:  "InstanceId",
		Value: p.InstanceId,
	}

	auth, err := aws.GetAuth(p.AccessKeyId, p.SecretAccessKey, "", time.Now())
	if err != nil {
		return nil, err
	}
	cw, err := cloudwatch.NewCloudWatch(auth, region.CloudWatchServicepoint)

	stat := make(map[string]float64)

	stat["usage"], err = GetLastPointAverage(cw, dimension, "CPUCreditUsage")
	if err != nil {
		return nil, err
	}

	stat["balance"], err = GetLastPointAverage(cw, dimension, "CPUCreditBalance")
	if err != nil {
		return nil, err
	}

	return stat, nil
}

func (n CPUCreditPlugin) GraphDefinition() map[string](mp.Graphs) {
	return graphdef
}

func main() {
	optRegion := flag.String("region", "", "AWS Region")
	optInstanceId := flag.String("instance-id", "", "Instance ID")
	optAccessKeyId := flag.String("access-key-id", "", "AWS Access Key ID")
	optSecretAccessKey := flag.String("secret-access-key", "", "AWS Secret Access Key")
	optTempfile := flag.String("tempfile", "", "Temp file name")
	flag.Parse()

	var cpucredit CPUCreditPlugin

	if *optRegion == "" || *optInstanceId == "" {
		cpucredit.Region = aws.InstanceRegion()
		cpucredit.InstanceId = aws.InstanceId()
	} else {
		cpucredit.Region = *optRegion
		cpucredit.InstanceId = *optInstanceId
	}

	cpucredit.AccessKeyId = *optAccessKeyId
	cpucredit.SecretAccessKey = *optSecretAccessKey

	helper := mp.NewMackerelPlugin(cpucredit)
	if *optTempfile != "" {
		helper.Tempfile = *optTempfile
	} else {
		helper.Tempfile = "/tmp/mackerel-plugin-cpucredit"
	}

	if os.Getenv("MACKEREL_AGENT_PLUGIN_META") != "" {
		helper.OutputDefinitions()
	} else {
		helper.OutputValues()
	}
}
