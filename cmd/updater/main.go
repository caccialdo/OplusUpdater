package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/Houvven/OplusUpdater/pkg/updater"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
)

func getStringFlag(cmd *cobra.Command, flagName string) string {
	flag, err := cmd.Flags().GetString(flagName)
	if err != nil {
		log.Fatalf("Error in %s: %v", flagName, err)
	}
	return flag
}

func getIntFlag(cmd *cobra.Command, flagName string) int {
	flag, err := cmd.Flags().GetInt(flagName)
	if err != nil {
		log.Fatalf("Error in %s: %v", flagName, err)
	}
	return flag
}

func getBoolFlag(cmd *cobra.Command, flagName string) bool {
	flag, err := cmd.Flags().GetBool(flagName)
	if err != nil {
		log.Fatalf("Error in %s: %v", flagName, err)
	}
	return flag
}

var rootCmd = &cobra.Command{
	Use:   "oplus-updater",
	Short: " Use Oplus official api to query OPlus,OPPO and Realme Mobile 's OS version update.",
	Run: func(cmd *cobra.Command, args []string) {
		//Get the value of the flag
		otaVer := getStringFlag(cmd, "ota-version")
		androidVer := getStringFlag(cmd, "android-version")
		colorOSVer := getStringFlag(cmd, "colorOS-version")
		zone := getStringFlag(cmd, "zone")
		mode := getIntFlag(cmd, "mode")
		proxy := getStringFlag(cmd, "proxy")
		scanCarrierIds := getBoolFlag(cmd, "scan-carrier-ids")

		do := func(carrierID string) {
			responseCipher, err := updater.QueryUpdater(&updater.Attribute{
				OtaVer:     otaVer,
				AndroidVer: androidVer,
				ColorOSVer: colorOSVer,
				Zone:       zone,
				Mode:       mode,
				ProxyStr:   proxy,
				CarrierID:  carrierID,
			})
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			cipherJson, err := json.Marshal(responseCipher)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			fmt.Println(string(pretty.Color(pretty.Pretty(cipherJson), nil)))
		}

		if scanCarrierIds {
			for i := range [256]int{} {
				carrierID := fmt.Sprintf("%08b", i)
				fmt.Printf("carrier id: %s\n", carrierID)
				do(carrierID)
				fmt.Println()
			}
		} else {
			do("")
		}
	},
}

func init() {
	otaVerBytes, _ := exec.Command("getprop", "ro.build.version.ota").Output()
	otaVer := strings.TrimSpace(string(otaVerBytes))

	rootCmd.Flags().StringP("ota-version", "o", otaVer, "OTA version (required), e.g., --ota-version=RMX3820_11.A.00_0000_000000000000")
	rootCmd.Flags().StringP("android-version", "a", "nil", "Android version (optional), e.g., --android-version=Android14")
	rootCmd.Flags().StringP("colorOS-version", "c", "nil", "ColorOS version (optional), e.g., --colorOS-version=ColorOS14.1.0")
	rootCmd.Flags().StringP("zone", "z", "CN", "Server zone: CN (default), EU or IN (optional), e.g., --zone=CN")
	rootCmd.Flags().IntP("mode", "m", 0, "Mode: 0 (stable, default) or 1 (testing), e.g., --mode=0")
	rootCmd.Flags().StringP("proxy", "p", "", "Proxy server, e.g., --proxy=type://@host:port or --proxy=type://user:password@host:port, support http and socks proxy")
	rootCmd.Flags().Bool("scan-carrier-ids", false, "Cycles through all possible carrier IDs, e.g., --scan-carrier-ids")

	if err := rootCmd.MarkFlagRequired("ota-version"); err != nil {
		os.Exit(1)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
