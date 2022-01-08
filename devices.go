package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"

	"github.com/danielpaulus/go-ios/ios"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// @Summary      Get logs for iOS device container
// @Description  Get logs by type
// @Tags         device-logs
// @Produce      json
// @Param        log_type path string true "Log Type"
// @Param        device_udid path string true "Device UDID"
// @Success      200 {object} SimpleResponseJSON
// @Failure      200 {object} SimpleResponseJSON
// @Router       /device-logs/{log_type}/{device_udid} [get]
func GetDeviceLogs(w http.ResponseWriter, r *http.Request) {

	// Get the parameters
	vars := mux.Vars(r)
	key := vars["log_type"]
	key2 := vars["device_udid"]

	log.WithFields(log.Fields{
		"event": "get_device_logs",
	}).Info("Attempting to get logs of type:" + key + " for device with udid:" + key2)

	// Execute the command to restart the container by container ID
	commandString := "tail -n 1000 ./logs/*" + key2 + "/" + key + ".log"
	cmd := exec.Command("bash", "-c", commandString)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.WithFields(log.Fields{
			"event": "get_device_logs",
		}).Error("Could not get logs of type:" + key + " for device with udid:" + key2)
		SimpleJSONResponse(w, "get_device_logs", "No logs of this type available for this container.", 200)
		return
	}
	SimpleJSONResponse(w, "get_device_logs", out.String(), 200)
}

func ReturnDeviceInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["device_udid"]

	// Open our jsonFile
	jsonFile, err := os.Open("./configs/config.json")

	// if os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}

	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	// we initialize the devices array
	var devices Devices

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'users' which we defined above
	json.Unmarshal(byteValue, &devices)

	w.Header().Set("Content-Type", "text/plain")

	// Loop over the devices and return info only on the device which UDID matches the path key
	for i := 0; i < len(devices.Devices); i++ {
		if devices.Devices[i].DeviceUDID == key {
			fmt.Fprintf(w, "Device Name: "+devices.Devices[i].DeviceName+"\n")
			fmt.Fprintf(w, "Appium Port: "+strconv.Itoa(devices.Devices[i].AppiumPort)+"\n")
			fmt.Fprintf(w, "Device OS version: "+devices.Devices[i].DeviceOSVersion+"\n")
			fmt.Fprintf(w, "Device UDID: "+devices.Devices[i].DeviceUDID+"\n")
			fmt.Fprintf(w, "WDA Mjpeg port: "+strconv.Itoa(devices.Devices[i].WdaMjpegPort)+"\n")
			fmt.Fprintf(w, "WDA Port: "+strconv.Itoa(devices.Devices[i].WdaPort)+"\n")
		}
	}
}

func GetConnectedIOSDevices(w http.ResponseWriter, r *http.Request) {
	deviceList, err := ios.ListDevices()
	if err != nil {
		http.Error(w, "Couldn't get iOS devices with go-ios or no devices connected to the machine. Error: "+err.Error(), http.StatusInternalServerError)
	}
	deviceValues, err := outputDetailedList(deviceList)
	fmt.Fprintf(w, deviceValues)
}

type detailsEntry struct {
	Udid           string
	ProductName    string
	ProductType    string
	ProductVersion string
}

func outputDetailedList(deviceList ios.DeviceList) (string, error) {
	result := make([]detailsEntry, len(deviceList.DeviceList))
	for i, device := range deviceList.DeviceList {
		udid := device.Properties.SerialNumber
		allValues, err := ios.GetValues(device)
		if err != nil {
			return "", errors.New("Failed getting device values")
		}
		result[i] = detailsEntry{udid, allValues.Value.ProductName, allValues.Value.ProductType, allValues.Value.ProductVersion}
	}
	return ConvertToJSONString(map[string][]detailsEntry{
		"deviceList": result,
	}), nil
}

func RegisterIOSDevice(w http.ResponseWriter, r *http.Request) {
	// Get the request json body and extract the device UDID and OS version
	requestBody, _ := ioutil.ReadAll(r.Body)
	device_udid := gjson.Get(string(requestBody), "device_udid")
	device_os_version := gjson.Get(string(requestBody), "device_os_version")
	device_name := gjson.Get(string(requestBody), "device_name")

	// Open the configuration json file
	jsonFile, err := os.Open("./configs/config.json")
	if err != nil {
		JSONError(w, "config_file_error", "Could not open the config.json file.", 500)
	}
	defer jsonFile.Close()

	// Read the configuration json file into byte array
	configJson, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		JSONError(w, "config_file_error", "Could not read the config.json file.", 500)
	}

	// Get the UDIDs of all devices registered in the config.json
	jsonDevicesUDIDs := gjson.Get(string(configJson), "devicesList.#.device_udid")

	//Loop over the devices UDIDs and return message if device is already registered
	// for _, udid := range jsonDevicesUDIDs.Array() {
	// 	if udid.String() == device_udid.String() {
	// 		JSONError(w, "device_registered", "The device with UDID: "+device_udid.String()+" is already registered.", 400)
	// 		return
	// 	}
	// }

	// Create the object for the new device
	var deviceInfo = Device{
		AppiumPort:      4841 + len(jsonDevicesUDIDs.Array()),
		DeviceName:      device_name.Str,
		DeviceOSVersion: device_os_version.String(),
		DeviceUDID:      device_udid.String(),
		WdaMjpegPort:    20101 + len(jsonDevicesUDIDs.Array()),
		WdaPort:         20001 + len(jsonDevicesUDIDs.Array())}

	// Append the new device object to the devicesList array
	updatedJSON, _ := sjson.Set(string(configJson), "devicesList.-1", deviceInfo)

	// Prettify the json so it looks good inside the file
	var prettyJSON bytes.Buffer
	json.Indent(&prettyJSON, []byte(updatedJSON), "", "  ")

	// Write the new json to the config.json file
	err = ioutil.WriteFile("./configs/config.json", []byte(prettyJSON.String()), 0644)
	if err != nil {
		JSONError(w, "config_file_error", "Could not write to the config.json file.", 400)
	}
}
