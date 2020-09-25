/*
copyright 2020 the Goployer authors

licensed under the apache license, version 2.0 (the "license");
you may not use this file except in compliance with the license.
you may obtain a copy of the license at

    http://www.apache.org/licenses/license-2.0

unless required by applicable law or agreed to in writing, software
distributed under the license is distributed on an "as is" basis,
without warranties or conditions of any kind, either express or implied.
see the license for the specific language governing permissions and
limitations under the license.
*/

package templates

const DeploymentSummary = `============================================================
Configurations Summary
============================================================
{{ .ConfigSummary }}
============================================================
{{- if eq (len .Stacks) 0 }}
 No stack selected
{{- else }}
Configurations of stacks
============================================================
{{- range $stack := .Stacks }}
{{ decorate "underline bold" "Stack" }}:	{{ $stack.Account }}
{{ decorate "underline bold" "Account" }}:	{{ $stack.Account }}
{{ decorate "underline bold" "Environment" }}:	{{ $stack.Env }}
{{ decorate "underline bold" "IAM Instance Profile" }}:	{{ $stack.IamInstanceProfile }}
{{- if gt (len $stack.Tags) 0 }}
{{ decorate "underline bold" "Tags" }}:	{{ joinString $stack.Tags "," }}
{{- end }}
{{- if gt (len $stack.AssumeRole) 0 }}
{{ decorate "underline bold" "Assume Role" }}:	{{ $stack.EbsOptimized }}
{{- end }}
{{ decorate "underline bold" "EBS Optimized" }}:	{{ $stack.EbsOptimized }}
{{ decorate "underline bold" "API Test Enabled" }}:	{{ $stack.APITestEnabled }}
{{ decorate "underline bold" "Capacity" }}
MINIMUM 	DESIRED 	MAXIMUM
{{ $stack.Capacity.Min }}	{{ $stack.Capacity.Desired }}	{{ $stack.Capacity.Max }}

{{- if gt (len $stack.BlockDevices) 0 }}
{{ decorate "underline bold" "Block Devices" }}
NAME	TYPE	SIZE	IOPS
{{- range $ebs := $stack.BlockDevices }}
{{ $ebs.DeviceName }}	{{ $ebs.VolumeType }}	{{ $ebs.VolumeSize }}	{{ $ebs.Iops }}
{{- end }}
{{- end }}

{{- if eq $stack.MixedInstancesPolicy.Enabled true }}
{{ decorate "underline bold" "Mixed Instance policy" }}
{{ decorate "bullet" "Override" }}: {{ joinString $stack.MixedInstancesPolicy.Override "," }}
{{ decorate "bullet" "On-Demand Percentage" }}: {{ $stack.MixedInstancesPolicy.OnDemandPercentage }}
{{ decorate "bullet" "Spot Allocation Strategy" }}: {{ $stack.MixedInstancesPolicy.SpotAllocationStrategy }}
{{ decorate "bullet" "Spot Instance Pools" }}: {{ $stack.MixedInstancesPolicy.SpotInstancePools }}
{{ decorate "bullet" "Spot Max Price" }}: {{ $stack.MixedInstancesPolicy.SpotMaxPrice }}
{{- end }}

{{ decorate "underline bold" "Region Configurations" }}
{{- range $index, $region := $stack.Regions }}
 {{- if or (eq (len .Region) 0) (eq $region.Region .Region) }}
{{ decorate "underline bold" "Region" }}: {{ $region.Region }}
{{ decorate "bullet" (decorate "bold" "Instance Type") }}: {{ $region.InstanceType }}
{{ decorate "bullet" (decorate "bold" "SSH Key") }}: {{ $region.SSHKey }}
{{ decorate "bullet" (decorate "bold" "VPC") }}: {{ $region.VPC }}
{{ decorate "bullet" (decorate "bold" "Use Public Subnets") }}: {{ $region.UsePublicSubnets }}
{{ decorate "bullet" (decorate "bold" "Detailed Monitoring Enabled") }}: {{ $region.DetailedMonitoringEnabled }}
{{- if (gt (len $region.AmiID) 0) }}
{{ decorate "bullet" (decorate "bold" "AMI ID") }}: {{ $region.AmiID }}
{{- end }}
{{- if (gt (len $region.SSHKey) 0) }}
{{ decorate "bullet" (decorate "bold" "SSH Key") }}: {{ $region.SSHKey }}
{{- end }}
{{- if (gt (len $region.SecurityGroups) 0) }}
{{ decorate "bullet" (decorate "bold" "Security Groups") }}: {{ joinString $region.SecurityGroups "," }}
{{- end }}
{{- if (gt (len $region.TargetGroups) 0) }}
{{ decorate "bullet" (decorate "bold" "Target Groups") }}: {{ joinString $region.TargetGroups "," }}
{{- end }}
{{- if (gt (len $region.LoadBalancers) 0) }}
{{ decorate "bullet" (decorate "bold" "Load Balancers") }}: {{ joinString $region.LoadBalancers "," }}
{{- end }}
{{- if (gt (len $region.HealthcheckLB) 0) }}
{{ decorate "bullet" (decorate "bold" "Healthcheck LB") }}: {{ $region.HealthcheckLB }}
{{- end }}
{{- if (gt (len $region.HealthcheckTargetGroup) 0) }}
{{ decorate "bullet" (decorate "bold" "Healthcheck TG") }}: {{ $region.HealthcheckTargetGroup }}
{{- end }}
{{- if (gt (len $region.AvailabilityZones) 0) }}
{{ decorate "bullet" (decorate "bold" "Availability Zones") }}: {{ joinString $region.AvailabilityZones "," }}
{{- end }}

 {{- end }}
{{- end }}
============================================================
{{- end }}
{{- end }}

`

const StatusResultTemplate = `{{decorate "bold" "Name"}}:	{{ .Summary.Name }}
{{decorate "bold" "Created Time"}}:	{{ .Summary.CreatedTime }}

{{decorate "capacity" ""}}{{decorate "underline bold" "Capacity"}}
MINIMUM 	DESIRED 	MAXIMUM
{{ .Summary.Capacity.Min }}	{{ .Summary.Capacity.Desired }}	{{ .Summary.Capacity.Max }}

{{decorate "instance_statistics" ""}}{{decorate "underline bold" "Instance Statistics"}}

{{- if eq (len .Summary.InstanceType) 0 }}
 No instance exists
{{- else }}
{{- range $k, $v := .Summary.InstanceType }}
 {{decorate "bullet" $k }}: {{ $v }}
{{- end }}
{{- end }}

{{decorate "tags" ""}}{{decorate "underline bold" "Tags"}}

{{- if eq (len .Summary.Tags) 0 }}
 No tag
{{- else }}
{{- range $result := .Summary.Tags }}
 {{decorate "bullet" $result }}
{{- end }}

{{decorate "security groups" ""}}{{decorate "underline bold" "Inbound Rules"}}
{{- if eq (len .Summary.IngressRules) 0 }}
 No inbound rules exist
{{- else }}
ID	PROTOCOL	FROM	TO	SOURCE	DESCRIPTION
{{- range $in := .Summary.IngressRules }}
 {{decorate "bullet" $in.ID }}	{{ $in.IPProtocol }}	{{ $in.FromPort }}	{{ $in.ToPort }}	{{ $in.IPRange }}	{{ $in.Description }}
{{- end }}
{{- end }}

{{decorate "security groups" ""}}{{decorate "underline bold" "Outbound Rules"}}
{{- if eq (len .Summary.EgressRules) 0 }}
 No outbound rules exist
{{- else }}
ID	PROTOCOL	FROM	TO	SOURCE	DESCRIPTION
{{- range $out := .Summary.EgressRules }}
 {{decorate "bullet" $out.ID }}	{{ $out.IPProtocol }}	{{ $out.FromPort }}	{{ $out.ToPort }}	{{ $out.IPRange }}	{{ $out.Description }}
{{- end }}
{{- end }}
{{- end }}

`

const APITestResultTemplate = `================API TEST RESULT================
{{- if eq (len .Metrics) 0 }}
 No metric exist
{{- else }}
Name:	{{ .Name }}
{{- range $metric := .Metrics }}
===============================================
API:	{{ $metric.URL }}
Method: {{ $metric.Method }}
DURATION	WAIT	REQUESTS	RATE	THROUGHPUT	SUCCESS
{{ round $metric.Data.Duration }}	{{ round $metric.Data.Wait }}	{{ $metric.Data.Requests }}	{{ roundNum $metric.Data.Rate }}	{{ roundNum $metric.Data.Throughput }}	{{ $metric.Data.Success }}

{{- if eq (len $metric.Data.StatusCodes) 0 }}
 No status codes exist
{{- else }}

Status	Status
{{- range $k, $v := $metric.Data.StatusCodes }}
 {{ decorate "bullet" $k }}	{{ $v }}
{{- end }}
{{- end }}

Latencies
TOTAL	MEAN	MAX	P95	P99
{{ round $metric.Data.Latencies.Total }}	{{ round $metric.Data.Latencies.Mean }}	{{ round $metric.Data.Latencies.Max }}	{{ round $metric.Data.Latencies.P95 }}	{{ round $metric.Data.Latencies.P99 }} 
{{- end }}
===============================================

{{- end }}

`
