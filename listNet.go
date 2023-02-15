package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
)

var ctx context.Context

type Net struct {
	SubscriptionName   string
	VirtualNetworkName string
	AddrSpace          string
}

var fileName string

func main() {
	flag.StringVar(&fileName, "fileName", "", "csv filename")
	flag.Parse()

	if fileName == "" {
		fmt.Fprintln(os.Stderr, "It's necessary to inform the filename with the option -fileName")
		os.Exit(1)
	}
	ctx = context.Background()
	cred, _ := azidentity.NewAzureCLICredential(nil)
	subs := getSubsIds(cred)
	var allVnets []Net
	for _, sub := range subs {
		vnets := getNets(cred, sub)
		allVnets = append(allVnets, vnets...)
	}
	file, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Comma = ';'
	writer.Write([]string{"SubscriptionName", "VirtualNetworkName", "AddrSpace"})
	for _, line := range allVnets {
		writer.Write([]string{line.SubscriptionName, line.VirtualNetworkName, line.AddrSpace})
	}
}

func ptrsToStrs(ptrs []*string) []string {
	var strs []string
	for _, ptr := range ptrs {
		strs = append(strs, *ptr)
	}
	return strs
}

func getNets(cred *azidentity.AzureCLICredential, sub map[string]string) []Net {
	var netList []Net
	netClient, err := armnetwork.NewVirtualNetworksClient(sub["id"], cred, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	pager := netClient.NewListAllPager(nil)
	for pager.More() {
		page, _ := pager.NextPage(ctx)
		for _, network := range page.Value {
			addr := strings.Join(ptrsToStrs(network.Properties.AddressSpace.AddressPrefixes), ", ")
			netList = append(netList, Net{*network.Name, sub["name"], addr})
		}
	}
	return netList
}

func getSubsIds(cred *azidentity.AzureCLICredential) []map[string]string {
	var subIds []map[string]string
	subClient, err := armsubscription.NewSubscriptionsClient(cred, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	pager := subClient.NewListPager(nil)
	for pager.More() {
		page, _ := pager.NextPage(ctx)
		for _, sub := range page.Value {
			m := map[string]string{
				"id":   *sub.SubscriptionID,
				"name": *sub.DisplayName,
			}
			subIds = append(subIds, m)
		}
	}
	return subIds
}
