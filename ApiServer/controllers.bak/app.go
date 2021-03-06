package controllers

import (
	"K8APITransform/ApiServer/Fti"
	"K8APITransform/ApiServer/lib"
	"K8APITransform/ApiServer/models"
	"crypto/tls"
	"encoding/json"
	"fmt"
	//"io/ioutil"
	//"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	//"github.com/GoogleCloudPlatform/kubernetes/pkg/fields"
	//"bufio"
	"io/ioutil"
	"log"
	//"net/url"
	"strconv"
	"strings"
	//"K8APITransform/ApiServer/backend"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"github.com/astaxie/beego"
	"io"
	"net/http"
	"os"
	//"path"
	//"time"
)

//store the relationship of podip and service ip
var serviceipmap = make(map[string]string)
var DockerBuilddeamon string

// Operations about App
type AppController struct {
	beego.Controller
}

// IntstrKind represents the stored type of IntOrString.
func NewIntOrStringFromInt(val int) models.IntOrString {
	return models.IntOrString{Kind: models.IntstrInt, IntVal: val}
}

// @Title CreateEnv
// @Description createEnv

// @router /createEnv [post]
func (a *AppController) CreateEnv() {
	ip := a.Ctx.Request.Header.Get("Authorization")
	var env models.AppEnv
	err := json.Unmarshal(a.Ctx.Input.RequestBody, &env)
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	err = env.Validate()
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	err = models.AddAppEnv(ip, &env)
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	detail := &models.Detail{Name: env.Name, Status: 1, NodeType: 1, Context: []models.Detail{}, Children: []models.Detail{}}
	detail.Cpu = env.Cpu
	detail.Memory = env.Memory
	detail.Storage = env.Storage
	detail.Children = append(detail.Children, models.Detail{
		Name:     "Nginx",
		Status:   1,
		NodeType: 2,
		Context: []models.Detail{
			models.Detail{
				Name:     "Node1",
				NodeType: 2,
			},
		},
	})
	num, err := strconv.Atoi(env.NodeNum)
	tomcat := models.Detail{Name: "tomcat", Status: 1, NodeType: 2, Context: []models.Detail{}, Children: []models.Detail{}}
	for i := 1; i <= num; i++ {
		tomcat.Context = append(tomcat.Context, models.Detail{
			Name:     "Node" + strconv.Itoa(i),
			NodeType: 3,
		})
	}
	detail.Children = append(detail.Children, tomcat)
	a.Data["json"] = detail
	a.ServeJson()
}

// @Title GetUploadWars
// @Description GetUploadWars

// @router /getuploadwars [get]
func (a *AppController) Getuploadwars() {
	ip := a.Ctx.Request.Header.Get("Authorization")
	username := ip
	dirhandle, err := os.Open("applications/" + username)
	//log.Println(dirname)
	//log.Println(reflect.TypeOf(dirhandle))
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	defer dirhandle.Close()

	//fis, err := ioutil.ReadDir(dir)
	fis, err := dirhandle.Readdir(0)
	//fis的类型为 []os.FileInfo
	//log.Println(reflect.TypeOf(fis))
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	result := []interface{}{}
	//遍历文件列表 (no dir inside) 每一个文件到要写入一个新的*tar.Header
	//var fi os.FileInfo
	for _, fi := range fis {

		//如果是普通文件 直接写入 dir 后面已经有了/
		filename := fi.Name()
		log.Println(filename)
		fileinfo := strings.Split(filename, "_")
		if fileinfo[len(fileinfo)-1] == "deploy" {
			filename = strings.TrimRight(filename, "_deploy")
			filename = strings.TrimRight(filename, ".war")
			fileinfo = strings.Split(filename, "-")
			version := fileinfo[len(fileinfo)-1]
			warname := strings.TrimSuffix(filename, "-"+version) + ".war"
			data := `{"id": 1,"name": "` + warname + `","nodeType": 0,"resource": [{"name": "app_version","value": "` + version + `"}]}`
			mapdata := map[string]interface{}{}
			json.Unmarshal([]byte(data), &mapdata)
			result = append(result, mapdata)
		}
		if err != nil {
			a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
			http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
			return
		}
	}
	a.Data["json"] = result
	a.ServeJson()
}

// @Title DeleteUploadWars
// @Description DeleteUploadWars

// @router /deleteuploadwars [delete]
func (a *AppController) DeleteUploadwars() {
	ip := a.Ctx.Request.Header.Get("Authorization")
	var warinfo = map[string]string{}

	err := json.Unmarshal(a.Ctx.Input.RequestBody, &warinfo)
	log.Println(warinfo)
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	prefix := strings.TrimSuffix(warinfo["warName"], ".war")
	err = os.RemoveAll("applications/" + ip + "/" + prefix + "-" + warinfo["version"] + ".war" + "_deploy")
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	err = os.RemoveAll("applications/" + ip + "/" + prefix + "-" + warinfo["version"] + ".war" + "_tar")
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	a.Data["json"] = map[string]string{"msg": "SUCCESS"}
	a.ServeJson()
}

// @Title checkUser
// @Description checkuser and store the ca.crt
// @router /checkuser [post]
func (a *AppController) Checkuser() {
	var clusterinfo = map[string]string{}
	err := json.Unmarshal(a.Ctx.Input.RequestBody, &clusterinfo)
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	//fmt.Println(clusterinfo)
	username := clusterinfo["userName"]
	//password := clusterinfo["password"]
	masterip := clusterinfo["masterIp"]
	cafile := clusterinfo["cacrt"]
	//cloudName := clusterinfo["cloudName"]
	//send the info to the node.js backend
	//client := &http.Client{}
	//data := url.Values{}
	//data.Add("userName", username)
	//data.Add("password", password)
	//data.Add("masterIp", masterip)
	//fmt.Println(data)
	//resp, err := client.PostForm("http://10.10.105.135:8800/user/checkAndUpdate", data)
	//if err != nil {
	//	a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
	//	http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
	//	return
	//}
	//defer resp.Body.Close()

	//body, _ := ioutil.ReadAll(resp.Body)
	//fmt.Println(string(body))
	//body = []byte("true")
	//if strings.Contains(string(body), "true") {
	//store the ca.crt into the file into certs/ip:port/ca.crt
	//create the ca.crt certs/ip:port/ca.crt
	//write the string in to the ca.crt
	data := []byte(cafile)
	filename := "certs/" + masterip + "/ca.crt"
	os.Mkdir("certs/"+masterip, 0777)
	err = ioutil.WriteFile(filename, data, 0666)
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}

	// add the info into the /etc/hosts
	//echo masterip username >> /etc/hosts
	//command := "echo " + masterip + " " + username + ">>
	filedata, err := ioutil.ReadFile("/etc/hosts")
	//fmt.Println(filedata)
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	lines := strings.Split(string(filedata), string(10))
	for _, v := range lines {
		fmt.Println(v)
	}
	//fmt.Println(lines)
	//fmt.Println(len(lines))
	//ret := make([]string, len(lines))
	ret := []string{}
	//has := 0
	for _, v := range lines {
		if v != "" && !strings.Contains(v, username) && !strings.Contains(v, masterip) {
			ret = append(ret, v)
			//fmt.Println(v)
		}
	}
	fmt.Println(strings.Join(ret, ",\n"))
	ret = append(ret, masterip+" "+username)
	back := strings.Join(ret, "\n")
	//fmt.Println(back)
	err = ioutil.WriteFile("/etc/hosts", []byte(back), 0777)
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	//bufio.NewReader()
	//Fti.Systemexec(command)
	_, err = models.EtcdClient.Get("/iptohost/"+masterip, false, false)
	if err != nil {
		models.EtcdClient.Create("/iptohost/"+masterip, username, 0)
	} else {
		models.EtcdClient.Update("/iptohost/"+masterip, username, 0)
	}
	a.Ctx.ResponseWriter.WriteHeader(200)
	//return
	//} else {
	//	a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
	//	http.Error(a.Ctx.ResponseWriter, `{"errorMessage: check fail}`, 406)
	//	return
	//}

}

// @Title upload war
// @Description upload

// @router /upload [post]
func (a *AppController) Upload() {
	//a.ParseForm()
	ip := a.Ctx.Request.Header.Get("Authorization")
	file, _, err := a.GetFile("filePath")
	version := a.GetString("version")
	appName := a.GetString("appName")
	log.Println(version)
	date := []byte(appName)
	date = date[0 : len(date)-4]
	//todo :use regx
	app_part := string(date)
	appName_tmp := app_part + "-" + version + ".war"

	username := ip
	//uploaddir := "applications/" + username + "/" + appName + "-" + version + "_deploy/"
	uploaddir := "applications/" + username + "/" + appName_tmp + "_deploy/"
	Fti.Createdir(uploaddir)
	//version := a.GetString("version")

	log.Println(version)
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	//f, err := os.OpenFile("applications/"+handle.Filename+version, os.O_WRONLY|os.O_CREATE, 0666)
	log.Println(uploaddir)
	//f, err := os.OpenFile(uploaddir+appName+"-"+version, os.O_WRONLY|os.O_CREATE, 0666)
	f, err := os.OpenFile(uploaddir+appName, os.O_WRONLY|os.O_CREATE, 0666)
	io.Copy(f, file)
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	defer f.Close()
	defer file.Close()
	a.Data["json"] = map[string]string{"msg": "SUCCESS"}
	a.ServeJson()
}

// @Title deploy
// @Description deploy
// @router /deploy [post]
func (a *AppController) Deploy() {
	namespace := "default"
	ip := a.Ctx.Request.Header.Get("Authorization")
	K8sBackend, err := models.NewBackendTLS(ip, "v1beta3")
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	deployReq := models.DeployRequest{}
	err = json.Unmarshal(a.Ctx.Input.RequestBody, &deployReq)
	log.Println(deployReq)
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	err = deployReq.Validate()
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	env, err := models.GetAppEnv(ip, deployReq.EnvName)
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}

	uploadfilename := deployReq.WarName
	//var uploadfilename string
	//if deployReq.AppVersion != "" {
	//	uploadfilename = deployReq.WarName + "-" + deployReq.AppVersion
	//} else {
	//	uploadfilename = deployReq.WarName
	//}

	username := ip
	//newimage := uploadfilename
	//newimage_part := strings.Split(uploadfilename, "-")[0]
	if deployReq.IsGreyUpdating == "0" {
		label := map[string]string{}
		label["env"] = deployReq.EnvName
		serviceslist, err := K8sBackend.Services(namespace).List(labels.SelectorFromSet(label))
		if err != nil {
			a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
			http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
			return
		}
		for _, v := range serviceslist.Items {
			//a.deleteapp(v.ObjectMeta.Labels["name"])
			K8sBackend.Applications(env.Name).Delete(v.ObjectMeta.Labels["name"])
		}
	}

	warName := uploadfilename
	newimage_name_temp := []byte(uploadfilename)
	newimage_name := string(newimage_name_temp[0 : len(newimage_name_temp)-4])
	//newimage_name := strings.Split(newimage_part, ".")[0]
	newimage_version := deployReq.AppVersion
	newimage := newimage_name + "-" + newimage_version + ".war"
	log.Println("newimagename:", newimage)
	//deployReq imagename string, uploaddir string) error
	//dockerdeamon := "unix:///var/run/docker.sock"
	//dockerdeamon := "http://10.211.55.10:2376"
	//dockerdeamon := DockerBuilddeamon
	dockerdeamon := "http://" + ip + ":2376"
	imageprefix := "k8master" + "reg:5000"

	//deployReq imagename string, uploaddir string) error
	//dockerdeamon := "unix:///var/run/docker.sock"
	baseimage := imageprefix + `\/apm-jre7-tomcat7:v4`
	//baseimage = env.JdkV + "-" + env.TomcatV
	//baseimage := "jre" + strconv(env.JdkV) + "-" + "tomcat" + strconv(env.TomcatV)
	newimage, err = Fti.Wartoimage(dockerdeamon, imageprefix, username, baseimage, newimage, warName)

	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	imagename := newimage
	log.Println("newimage:", imagename)

	//imagename := "7-jre-customize"
	//wartoimage
	//uploadimage
	//createapplication imagename = ""
	replicas, err := strconv.Atoi(env.NodeNum)
	app := models.AppCreateRequest{
		Name:    deployReq.WarName,
		Version: deployReq.AppVersion,
		Ports: []models.Port{
			models.Port{
				Port:       8080,
				TargetPort: 8080,
				Protocol:   "TCP",
			},
		},
		Replicas: replicas,
		ContainerPort: []models.Containerport{
			models.Containerport{
				Port:     8080,
				Protocol: "TCP",
			},
		},
		Cpu:            env.Cpu,
		Memory:         env.Memory,
		Storage:        env.Storage,
		Containername:  env.Name,
		Containerimage: imagename,
	}

	//Todo: get the se according to the label if err==nil the se already exist
	label := map[string]string{}
	label["env"] = deployReq.EnvName
	label["name"] = deployReq.EnvName + "-" + app.Name + "-" + app.Version
	selist, err := K8sBackend.Services(namespace).List(labels.SelectorFromSet(label))
	if len(selist.Items) != 0 {
		//already exist
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{errorMessage: the service already exist}`, 406)
		return
	}

	detail, err := K8sBackend.Applications(env.Name).Create(app)
	if err != nil {
		//a.deleteapp(app.Name + "-" + app.Version)
		K8sBackend.Applications(env.Name).Delete(app.Name + "-" + app.Version)
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	env.Used++
	err = models.UpdateAppEnv(ip, env.Name, env)
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}

	//time.After will return a new channel every time attention to the position of time.After
	a.Data["json"] = detail
	a.ServeJson()
}

// @Title get partDetails
// @Description get partDetails

// @router /test [get]
func (a *AppController) Test() {
	namespace := "default"
	ip := a.Ctx.Request.Header.Get("Authorization")
	K8sBackend, _ := models.NewBackendTLS(ip, "v1beta3")
	label := map[string]string{}
	label["env"] = "test"
	serviceslist, err := K8sBackend.Services(namespace).List(labels.SelectorFromSet(label))
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	a.Data["json"] = serviceslist
	a.ServeJson()
}

// @Title get partDetails
// @Description get partDetails
// @router /partDetails [get]
func (a *AppController) PartDetails() {
	//a.ParseForm()
	ip := a.Ctx.Request.Header.Get("Authorization")
	K8sBackend, err := models.NewBackendTLS(ip, "v1beta3")
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	//K8sBackend := a.GetSession("backend").(*models.Backend)
	envName := a.GetString("envName")
	log.Println(envName)
	detail, err := K8sBackend.Applications(envName).List()
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	//detail := a.getdetails(env)
	a.Data["json"] = detail
	a.ServeJson()

}

//func (a *AppController) getdetails(env *models.AppEnv) *models.Detail {
//	namespace := "default"
//	label := map[string]string{}
//	label["env"] = env.Name
//	serviceslist, err := K8sBackend.Services(namespace).List(labels.SelectorFromSet(label))
//	if err != nil {
//		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
//		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
//		return nil
//	}

//	//url := "http://" + models.KubernetesIp + ":8080/api/v1/namespaces/" + namespace + "/services" + "?labelSelector=env%3D" + env.Name
//	//log.Println(url)
//	//rsp, _ := http.Get(url)
//	//var serviceslist models.ServiceList
//	//body, _ := ioutil.ReadAll(rsp.Body)
//	//json.Unmarshal(body, &serviceslist)

//	//url = "http://" + models.KubernetesIp + ":8080/api/v1/namespaces/" + namespace + "/pods" + "?labelSelector=env%3D" + env.Name
//	//log.Println(url)
//	//rsp, _ = http.Get(url)
//	//var podslist models.PodList
//	//body, _ = ioutil.ReadAll(rsp.Body)
//	//json.Unmarshal(body, &podslist)
//	podslist, err := K8sBackend.Pods(namespace).List(labels.SelectorFromSet(label), nil)

//	detail := &models.Detail{Name: env.Name, Status: 1, NodeType: 1, Context: []models.Detail{}, Children: []models.Detail{}}
//	detail.Children = append(detail.Children, models.Detail{
//		Name:     "Nginx",
//		Status:   1,
//		NodeType: 2,
//		Context: []models.Detail{
//			models.Detail{
//				Name:     "Node1",
//				NodeType: 2,
//			},
//		},
//	})
//	tomcat := models.Detail{Name: "tomcat", Status: 1, NodeType: 2, Context: []models.Detail{}, Children: []models.Detail{}}
//	if len(podslist.Items) == 0 {
//		num, _ := strconv.Atoi(env.NodeNum)
//		for k := 0; k < num; k++ {
//			//names := strings.Split(v.ObjectMeta.Labels["name"], "-")
//			tomcat.Context = append(tomcat.Context, models.Detail{
//				Name:     "Node" + strconv.Itoa(k+1),
//				NodeType: 3,
//			})
//		}
//	} else {
//		for k, v := range podslist.Items {
//			status := 0
//			if v.Status.Phase == api.PodRunning {
//				status = 1
//			}
//			//names := strings.Split(v.ObjectMeta.Labels["name"], "-")
//			tomcat.Context = append(tomcat.Context, models.Detail{
//				Name:       "Node" + strconv.Itoa(k+1),
//				AppVersion: v.ObjectMeta.Labels["name"],
//				Status:     status,
//				NodeType:   3,
//			})
//		}
//	}
//	apps := []models.Detail{}
//	for _, v := range serviceslist.Items {
//		//names := strings.Split(v.ObjectMeta.Labels["name"], "-")
//		apps = append(apps, models.Detail{
//			Name:     v.ObjectMeta.Labels["name"],
//			NodeType: 4,
//			Status:   1,
//			Resource: []models.Detail{models.Detail{Name: "IP", Value: v.Spec.ClusterIP + ":8080"}},
//		})
//	}
//	tomcat.Children = append(tomcat.Children, models.Detail{
//		Name:     "应用",
//		NodeType: 3,
//		Context:  []models.Detail{},
//	})
//	tomcat.Children[0].Context = append(tomcat.Children[0].Context, apps...)
//	detail.Children = append(detail.Children, tomcat)
//	return detail
//}

// @Title get Details
// @Description get Details

// @router /details [get]
func (a *AppController) Details() {
	ip := a.Ctx.Request.Header.Get("Authorization")
	K8sBackend, err := models.NewBackendTLS(ip, "v1beta3")
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	envs, err := models.GetAllAppEnv(ip)
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	response := []*models.Detail{}
	for _, v := range envs {
		detail, _ := K8sBackend.Applications(v.Name).List()
		response = append(response, detail)
	}
	a.Data["json"] = response
	a.ServeJson()
}

// @Title restartApp
// @Description restartApp

// @router /restartApp [post]
func (a *AppController) RestartApp() {
	ip := a.Ctx.Request.Header.Get("Authorization")
	K8sBackend, err := models.NewBackendTLS(ip, "v1beta3")
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	//namespace := "default"
	req := map[string]string{}
	err = json.Unmarshal(a.Ctx.Input.RequestBody, &req)
	//log.Println(deployReq)
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	log.Println(req["appName"])
	if _, exist := req["appName"]; exist == false {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+"request not has appName "+`"}`, 406)
		return
	}
	//appName := req["appName"]
	//appName := app.Name + "-" + app.Version
	err = K8sBackend.Applications(req["envName"]).Restart(req["appName"])
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	a.Data["json"] = map[string]string{"msg": "SUCCESS"}
	a.ServeJson()
}

// @Title getEnv
// @Description getEnv

// @router /getEnv/:envname [get]
func (a *AppController) GetEnv() {
	ip := a.Ctx.Request.Header.Get("Authorization")
	//K8sBackend := models.NewBackendTLS(ip, "v1beta3")
	name := a.Ctx.Input.Param(":envname")
	env, err := models.GetAppEnv(ip, name)
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	a.Data["json"] = env
	a.ServeJson()
}

// @Title deleteEnv
// @Description deleteEnv

// @router /deleteEnv/:envname [delete]
func (a *AppController) DeleteEnv() {
	ip := a.Ctx.Request.Header.Get("Authorization")
	K8sBackend, err := models.NewBackendTLS(ip, "v1beta3")
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	name := a.Ctx.Input.Param(":envname")
	err = models.DeleteAppEnv(ip, name)

	//env, err := models.GetAppEnv(name)
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	K8sBackend.Applications(name).DeleteAll()
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	a.Data["json"] = map[string]string{"msg": "SUCCESS"}
	a.ServeJson()
}

// @Title getpodsip
// @Description getpodsip
// @router /podsip/:sename [get]
func (a *AppController) Getpodsip() {
	ip := a.Ctx.Request.Header.Get("Authorization")
	K8sBackend, err := models.NewBackendTLS(ip, "v1beta3")
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	sename := a.Ctx.Input.Param(":sename")
	log.Println(sename)
	iplist, err := K8sBackend.Podip(ip, sename)
	if err != nil {
		log.Println(err.Error())
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	a.Data["json"] = iplist
	a.ServeJson()

}

// @Title getservice
// @Description getservice
// @router /serviceip/:podip [get]
func (a *AppController) Getseip() {
	ip := a.Ctx.Request.Header.Get("Authorization")
	podip := a.Ctx.Input.Param(":podip")
	fmt.Println(podip)
	//todo:watch the etcd
	//seip := serviceipmap[podip]
	seip, err := models.GetPodtoSe(ip, podip)
	if err != nil {
		log.Println(err.Error())
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	a.Data["json"] = seip
	a.ServeJson()

	//return nil
}

// @Title ScaleApp
// @Description Scale app
// @Param       namespaces      path    string  true            "The key for namespaces"
// @Param       service         path    string  true            "The key for name"
// @Param       body            body    models.AppUpgradeRequest         true           "body for user content"
// @Success 200 {string} "scale success"
// @Failure 403 body is empty
// @router /scaleApp [put]
func (a *AppController) Scale() {
	//namespace := "default"
	ip := a.Ctx.Request.Header.Get("Authorization")
	K8sBackend, err := models.NewBackendTLS(ip, "v1beta3")
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	var appScale models.AppScale
	fmt.Println(string(a.Ctx.Input.RequestBody))
	err = json.Unmarshal(a.Ctx.Input.RequestBody, &appScale)

	fmt.Println(appScale.Num)
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	replicas, _ := strconv.Atoi(appScale.Num)
	datails, err := K8sBackend.Applications(appScale.EnvName).Update(appScale.Name, replicas)
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	a.Data["json"] = datails
	a.ServeJson()
}

// @Title createApp
// @Description create app
// @Param	namespaces	path 	string	true		"The key for namespaces"
// @Param	service		path 	string	true		"The key for name"
// @Success 200 {string} "create success"
// @Failure 403 body is empty
// @router /deleteApp [delete]
func (a *AppController) DeleteApp() {
	//namespace := "default"
	//service := a.Ctx.Input.Param(":service")
	ip := a.Ctx.Request.Header.Get("Authorization")
	K8sBackend, err := models.NewBackendTLS(ip, "v1beta3")
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	var app = map[string]string{}
	err = json.Unmarshal(a.Ctx.Input.RequestBody, &app)
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	//log.Println(appScale.Num)

	if _, exist := app["appName"]; exist == false {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+"not send appName"+`"}`, 406)
		return
	}
	appName := app["appName"]
	envName := strings.Split(appName, "-")[0]
	//a.deleteapp(ip, appName)
	K8sBackend.Applications(envName).Delete(appName)
	//re := map[string]interface{}{}
	//re["delete rc"] = result
	//delete(models.Appinfo[namespace], service)
	a.Data["json"] = map[string]string{"msg": "SUCCESS"}
	a.ServeJson()

}

func (a *AppController) deleteapp(ip string, appName string) {
	namespace := "default"
	//K8sBackend := models.NewBackendTLS(ip, "v1beta3")
	//appName = strings.ToLower(appName)
	//appName = strings.Replace(appName, ".", "", -1)
	lib.Sendapi("DELETE", ip, "8080", "v1", []string{"namespaces", namespace, "services", appName}, []byte{})
	//re["delete service"] = result
	lib.Sendapi("DELETE", ip, "8080", "v1", []string{"namespaces", namespace, "replicationcontrollers", appName}, []byte{})
}

// @Title get events
// @Description get events

// @router /events [get]
func (a *AppController) GetEvents() {
	//namespace := "default"
	ip := a.Ctx.Request.Header.Get("Authorization")
	fmt.Println(ip)
	K8sBackend, err := models.NewBackendTLS(ip, "v1beta3")
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	//K8sBackend := a.GetSession("backend").(*models.Backend)
	//se := fields.SelectorFromSet(map[string]string{"involvedObject.kind": "Pod"})
	//fmt.Println(se)
	data, err := K8sBackend.Events("default").List(nil, nil)
	//data, err := K8sBackend.Services("default").List(nil)
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	a.Data["json"] = data
	a.ServeJson()
}

//// @Title get status
//// @Description get status

//// @router /status [get]
//func (a *AppController) GetStatus() {
//	//namespace := "default"
//	ip := a.Ctx.Request.Header.Get("Authorization")
//	K8sBackend, _ := models.NewBackendTLS(ip, "v1beta3")
//	se := fields.SelectorFromSet(map[string]string{"involvedObject.kind": "Pod"})
//	fmt.Println(se)
//	data, _ := K8sBackend.Nodes().List(nil, nil)
//	for _, node := range data.Items {
//		if node.Status.Conditions[0].Type != api.NodeReady {
//			a.Data["json"] = map[string]string{"msg": "Not Ready"}
//			a.ServeJson()
//			return
//		}
//	}
//	a.Data["json"] = map[string]string{"msg": "Ready"}
//	a.ServeJson()
//}

// @Title get node status
// @Description get node status
// @router /nodestatus [get]
func (a *AppController) NodeStatus() {
	ip := a.Ctx.Request.Header.Get("Authorization")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	request, err := http.NewRequest("GET", "https://"+ip+":50000/api/cluster/status", nil)
	request.Header.Set("token", "qwertyuiopasdfghjklzxcvbnm1234567890")
	response, err := client.Do(request)
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	defer response.Body.Close()
	body, _ := ioutil.ReadAll(response.Body)
	//fmt.Println(string(body))
	var w = map[string]interface{}{}
	json.Unmarshal(body, &w)
	a.Data["json"] = w
	a.ServeJson()
}

// @Title get app status
// @Description get app status
// @router /appstatus [post]
func (a *AppController) AppStatus() {
	ip := a.Ctx.Request.Header.Get("Authorization")
	K8sBackend, err := models.NewBackendTLS(ip, "v1beta3")
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	var input = map[string]string{}
	err = json.Unmarshal(a.Ctx.Input.RequestBody, &input)
	if err != nil {
		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
		http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
		return
	}
	//envName := input["envName"]
	appName := input["appName"]
	fmt.Println(appName)
	podslist, err := K8sBackend.Pods("default").List(labels.SelectorFromSet(map[string]string{"name": appName}), nil)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	var output = []interface{}{}
	for _, pod := range podslist.Items {
		if len(pod.Status.ContainerStatuses) == 0 || pod.Status.ContainerStatuses[0].ContainerID == "" {
			continue
		}
		id := pod.Status.ContainerStatuses[0].ContainerID

		id = strings.TrimPrefix(id, "docker://")
		fmt.Println(id)
		request, err := http.NewRequest("GET", "https://"+ip+":50000/api/container/status", nil)
		request.Header.Set("token", "qwertyuiopasdfghjklzxcvbnm1234567890")
		request.Header.Set("container", id)
		response, err := client.Do(request)
		if err != nil {
			a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json")
			http.Error(a.Ctx.ResponseWriter, `{"errorMessage":"`+err.Error()+`"}`, 406)
			return
		}
		defer response.Body.Close()
		body, _ := ioutil.ReadAll(response.Body)
		//fmt.Println(string(body))
		var w = map[string]interface{}{}
		json.Unmarshal(body, &w)
		output = append(output, w)
	}

	a.Data["json"] = output
	a.ServeJson()
}

//// @Title get all apps
//// @Description get all apps
//// @Param	namespaces	path 	string	true		"The key for namespaces"
//// @Success 200 {string} "get success"
//// @router / [get]
//func (a *AppController) GetAll() {
//	namespaces := a.Ctx.Input.Param(":namespaces")

//	status, result := lib.Sendapi("GET", models.KubernetesIp, "8080", "v1", []string{"namespaces", namespaces, "services"}, []byte{})
//	responsebodyK8s, _ := json.Marshal(result)
//	if status != 200 {
//		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
//		a.Ctx.ResponseWriter.WriteHeader(status)
//		log.Fprintln(a.Ctx.ResponseWriter, string(responsebodyK8s))
//		return
//	}

//	var appListK8s models.ServiceList //service -> app

//	var appList models.AppGetAllResponse
//	var app models.AppGetAllResponseItem

//	appList.Items = make([]models.AppGetAllResponseItem, 0, 60)

//	json.Unmarshal([]byte(responsebodyK8s), &appListK8s)

//	for index := 0; index < len(appListK8s.Items); index++ {
//		app = models.AppGetAllResponseItem{
//			Name: appListK8s.Items[index].ObjectMeta.Name,
//		}
//		appList.Items = append(appList.Items, app)
//	}

//	//appList.Kind = appListK8s.TypeMeta.Kind
//	appList.Kind = "AppGetAllResponse"

//	responsebody, _ := json.Marshal(appList)

//	a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
//	a.Ctx.ResponseWriter.WriteHeader(status)
//	log.Fprintln(a.Ctx.ResponseWriter, string(responsebody))

//	//a.Data["json"] = map[string]string{"status": "getall success"}
//	//a.ServeJson()
//}

//// @Title Get App
//// @Description get app by name and namespace
//// @Param	namespaces	path 	string	true		"The key for namespaces"
//// @Param	service		path 	string	true		"The key for name"
//// @Success 200 {string} "get success"
//// @router /:service [get]
//func (a *AppController) Get() {
//	namespaces := a.Ctx.Input.Param(":namespaces")
//	name := a.Ctx.Input.Param(":service")

//	status, result := lib.Sendapi("GET", models.KubernetesIp, "8080", "v1", []string{"namespaces", namespaces, "services", name}, []byte{})
//	responsebodyK8s, _ := json.Marshal(result)

//	if status != 200 {
//		a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
//		a.Ctx.ResponseWriter.WriteHeader(status)
//		log.Fprintln(a.Ctx.ResponseWriter, string(responsebodyK8s))
//		return
//	}

//	var appK8s models.Service //service -> app
//	json.Unmarshal([]byte(responsebodyK8s), &appK8s)

//	var app = models.AppGetResponse{
//		Kind:              "AppGetResponse",
//		Name:              appK8s.ObjectMeta.Name,
//		Namespace:         appK8s.ObjectMeta.Namespace,
//		CreationTimestamp: appK8s.ObjectMeta.CreationTimestamp,
//		Labels:            appK8s.ObjectMeta.Labels,
//		Spec:              appK8s.Spec,
//		Status:            appK8s.Status,
//	}
//	responsebody, _ := json.Marshal(app)

//	a.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
//	a.Ctx.ResponseWriter.WriteHeader(status)
//	log.Fprintln(a.Ctx.ResponseWriter, string(responsebody))

//	//a.Data["json"] = map[string]string{"status": "get success"}
//	//a.ServeJson()
//}

//// @Title createApp
//// @Description create app
//// @Param	namespaces	path 	string	true		"The key for namespaces"
//// @Param	service		path 	string	true		"The key for name"
//// @Success 200 {string} "create success"
//// @Failure 403 body is empty
//// @router /:service [delete]
//func (a *AppController) Deleteapp() {
//	namespace := a.Ctx.Input.Param(":namespaces")
//	service := a.Ctx.Input.Param(":service")
//	re := map[string]interface{}{}
//	_, result := lib.Sendapi("DELETE", models.KubernetesIp, "8080", "v1", []string{"namespaces", namespace, "services", service}, []byte{})
//	re["delete service"] = result
//	url := "http://" + models.KubernetesIp + ":8080/api/v1/namespaces/" + namespace + "/replicationcontrollers" + "?labelSelector=name%3D" + service
//	//log.Println(url)
//	rsp, _ := http.Get(url)
//	var rclist models.ReplicationControllerList
//	//var oldrc models.ReplicationController
//	body, _ := ioutil.ReadAll(rsp.Body)
//	//log.Println(string(body))
//	json.Unmarshal(body, &rclist)
//	//log.Println(rclist.Items[0].Spec)
//	if len(rclist.Items) == 0 {
//		a.Ctx.ResponseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
//		a.Ctx.ResponseWriter.WriteHeader(406)
//		log.Fprintln(a.Ctx.ResponseWriter, string("service with no rc"))
//		return
//	}
//	oldrc := rclist.Items[0]
//	oldrc.TypeMeta.Kind = "ReplicationController"
//	oldrc.TypeMeta.APIVersion = "v1"
//	oldrc.Spec.Replicas = 0
//	body, _ = json.Marshal(oldrc)
//	log.Println(string(body))
//	_, result = lib.Sendapi("PUT", models.KubernetesIp, "8080", "v1", []string{"namespaces", namespace, "replicationcontrollers", oldrc.ObjectMeta.Name}, body)
//	re["delete pod"] = result
//	//time.Sleep(5 * time.Second)

//	_, result = lib.Sendapi("DELETE", models.KubernetesIp, "8080", "v1", []string{"namespaces", namespace, "replicationcontrollers", oldrc.ObjectMeta.Name}, []byte{})
//	re["delete rc"] = result
//	delete(models.Appinfo[namespace], service)
//	a.Data["json"] = re
//	a.ServeJson()

//}

//// @Title get App state
//// @Description get App state
//// @Param	namespaces	path 	string	true		"The key for namespaces"
//// @Success 200 {string} "get App state success"
//// @Failure 403 body is empty
//// @router /:service/state [get]
//func (a *AppController) Getstate() {
//	namespace := a.Ctx.Input.Param(":namespaces")
//	service := a.Ctx.Input.Param(":service")
//	url := "http://" + models.KubernetesIp + ":8080/api/v1/namespaces/" + namespace + "/pods" + "?labelSelector=name%3D" + service
//	//log.Println(url)
//	rsp, _ := http.Get(url)

//	var rclist models.PodList
//	body, _ := ioutil.ReadAll(rsp.Body)
//	json.Unmarshal(body, &rclist)
//	log.Println(rclist.Items)
//	var res = map[models.PodPhase]int{}
//	for _, v := range rclist.Items {
//		res[v.Status.Phase]++
//	}
//	a.Data["json"] = res
//	a.ServeJson()
//}

//// @Title stop app
//// @Description stop app
//// @Param	namespaces	path 	string	true		"The key for namespaces"
//// @Param	service		path 	string	true		"The key for name"
//// @Success 200 {string} "stop success"
//// @Failure 403 body is empty
//// @router /:service/stop [get]
//func (a *AppController) Stop() {
//	namespace := a.Ctx.Input.Param(":namespaces")
//	service := a.Ctx.Input.Param(":service")

//	_, exist := models.Appinfo[namespace]
//	if !exist {
//		a.Ctx.ResponseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
//		a.Ctx.ResponseWriter.WriteHeader(406)
//		log.Fprintln(a.Ctx.ResponseWriter, `{"error":"no namespace`+namespace+`"}`)
//		return
//	}
//	_, exist = models.Appinfo[namespace][service]
//	if !exist {
//		a.Ctx.ResponseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
//		a.Ctx.ResponseWriter.WriteHeader(406)
//		log.Fprintln(a.Ctx.ResponseWriter, `{"error":"no service`+service+`"}`)
//		return
//	}
//	if models.Appinfo[namespace][service].Status == 0 {
//		a.Ctx.ResponseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
//		a.Ctx.ResponseWriter.WriteHeader(406)
//		log.Fprintln(a.Ctx.ResponseWriter, `{"error":"service `+service+` has already been stopped"}`)
//		return
//	}
//	url := "http://" + models.KubernetesIp + ":8080/api/v1/namespaces/" + namespace + "/replicationcontrollers" + "?labelSelector=name%3D" + service
//	//log.Println(url)
//	rsp, _ := http.Get(url)
//	var rclist models.ReplicationControllerList
//	//var oldrc models.ReplicationController
//	body, _ := ioutil.ReadAll(rsp.Body)
//	//log.Println(string(body))
//	json.Unmarshal(body, &rclist)
//	//log.Println(rclist.Items[0].Spec)
//	if len(rclist.Items) == 0 {
//		a.Ctx.ResponseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
//		a.Ctx.ResponseWriter.WriteHeader(406)
//		log.Fprintln(a.Ctx.ResponseWriter, string("service with no rc"))
//		return
//	}
//	oldrc := rclist.Items[0]
//	oldrc.TypeMeta.Kind = "ReplicationController"
//	oldrc.TypeMeta.APIVersion = "v1"
//	oldrc.Spec.Replicas = 0
//	body, _ = json.Marshal(oldrc)
//	log.Println(string(body))
//	status, result := lib.Sendapi("PUT", models.KubernetesIp, "8080", "v1", []string{"namespaces", namespace, "replicationcontrollers", oldrc.ObjectMeta.Name}, body)
//	log.Println(status)
//	if status != 200 {
//		a.Ctx.ResponseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
//		a.Ctx.ResponseWriter.WriteHeader(406)
//		log.Fprintln(a.Ctx.ResponseWriter, result)
//		return
//	} else {
//		models.Appinfo[namespace][service].Status = 0
//	}
//	a.Data["json"] = map[string]string{"messages": "start service successfully"}
//	a.ServeJson()
//}

//// @Title start app
//// @Description start app
//// @Param	namespaces	path 	string	true		"The key for namespaces"
//// @Param	service		path 	string	true		"The key for name"
//// @Success 200 {string} "start success"
//// @Failure 403 body is empty
//// @router /:service/start [get]
//func (a *AppController) Start() {
//	namespace := a.Ctx.Input.Param(":namespaces")
//	service := a.Ctx.Input.Param(":service")
//	_, exist := models.Appinfo[namespace]
//	if !exist {
//		a.Ctx.ResponseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
//		a.Ctx.ResponseWriter.WriteHeader(406)
//		log.Fprintln(a.Ctx.ResponseWriter, "no namespace "+namespace)
//		return
//	}
//	_, exist = models.Appinfo[namespace][service]
//	if !exist {
//		a.Ctx.ResponseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
//		a.Ctx.ResponseWriter.WriteHeader(406)
//		log.Fprintln(a.Ctx.ResponseWriter, "no service"+service)
//		return
//	}
//	if models.Appinfo[namespace][service].Status == 1 {
//		a.Ctx.ResponseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
//		a.Ctx.ResponseWriter.WriteHeader(406)
//		log.Fprintln(a.Ctx.ResponseWriter, "service "+service+" has already been started")
//		return
//	}
//	url := "http://" + models.KubernetesIp + ":8080/api/v1/namespaces/" + namespace + "/replicationcontrollers" + "?labelSelector=name%3D" + service
//	//log.Println(url)
//	rsp, _ := http.Get(url)
//	var rclist models.ReplicationControllerList
//	//var oldrc models.ReplicationController
//	body, _ := ioutil.ReadAll(rsp.Body)
//	//log.Println(string(body))
//	json.Unmarshal(body, &rclist)
//	//log.Println(rclist.Items[0].Spec)
//	if len(rclist.Items) == 0 {
//		a.Ctx.ResponseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
//		a.Ctx.ResponseWriter.WriteHeader(406)
//		log.Fprintln(a.Ctx.ResponseWriter, "service "+service+"with no rc")
//		return
//	}
//	oldrc := rclist.Items[0]
//	oldrc.TypeMeta.Kind = "ReplicationController"
//	oldrc.TypeMeta.APIVersion = "v1"
//	oldrc.Spec.Replicas = models.Appinfo[namespace][service].Replicas
//	body, _ = json.Marshal(oldrc)
//	log.Println(string(body))
//	status, result := lib.Sendapi("PUT", models.KubernetesIp, "8080", "v1", []string{"namespaces", namespace, "replicationcontrollers", oldrc.ObjectMeta.Name}, body)

//	if status != 200 {
//		a.Ctx.ResponseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
//		a.Ctx.ResponseWriter.WriteHeader(406)
//		log.Fprintln(a.Ctx.ResponseWriter, result)
//		return
//	} else {
//		models.Appinfo[namespace][service].Status = 1
//	}
//	a.Data["json"] = map[string]string{"messages": "start service successfully"}
//	a.ServeJson()
//}

//// @Title UpgradeApp
//// @Description Upgrade app
//// @Param	namespaces	path 	string	true		"The key for namespaces"
//// @Param	service		path 	string	true		"The key for name"
//// @Param	body		body 	models.AppUpgradeRequest	 true		"body for user content"
//// @Success 200 {string} "upgrade success"
//// @Failure 403 body is empty
//// @router /:service/upgrade [put]
//func (a *AppController) Upgrade() {
//	namespace := a.Ctx.Input.Param(":namespaces")
//	service := a.Ctx.Input.Param(":service")
//	var upgradeRequest models.AppUpgradeRequest
//	err := json.Unmarshal(a.Ctx.Input.RequestBody, &upgradeRequest)
//	log.Println(upgradeRequest.Containerimage)
//	if err != nil {
//		a.Ctx.ResponseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
//		a.Ctx.ResponseWriter.WriteHeader(406)
//		log.Fprintln(a.Ctx.ResponseWriter, err)
//		return
//	}
//	image := ""
//	//log.Println("%v", []byte(upgradeRequest.Warpath))
//	if upgradeRequest.Warpath == "" {
//		////
//		image = upgradeRequest.Containerimage
//	} else {
//		image = "" //war to image
//	}
//	//log.Println(image)
//	url := "http://" + models.KubernetesIp + ":8080/api/v1/namespaces/" + namespace + "/replicationcontrollers" + "?labelSelector=name%3D" + service
//	//log.Println(url)
//	rsp, err := http.Get(url)
//	var rclist models.ReplicationControllerList
//	//var oldrc models.ReplicationController
//	body, _ := ioutil.ReadAll(rsp.Body)
//	//log.Println(string(body))
//	json.Unmarshal(body, &rclist)
//	//log.Println(rclist.Items[0].Spec)
//	if len(rclist.Items) == 0 {
//		a.Ctx.ResponseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
//		a.Ctx.ResponseWriter.WriteHeader(406)
//		log.Fprintln(a.Ctx.ResponseWriter, "service "+service+"with no rc")
//		return
//	}
//	oldrc := rclist.Items[0]
//	oldrc.TypeMeta.Kind = "ReplicationController"
//	oldrc.TypeMeta.APIVersion = "v1"
//	//log.Println(rclist.Items[0])
//	//log.Println(oldrc.Spec.Template)
//	//var newrc ReplicationController
//	//log.Println(strings.Split(oldrc.ObjectMeta.Name, "-"))
//	oldversion, _ := strconv.Atoi(strings.Split(oldrc.ObjectMeta.Name, "-")[1])
//	newversion := service + "-" + strconv.Itoa(oldversion+1)

//	containers := []models.Container{
//		models.Container{
//			Name:  upgradeRequest.Containername,
//			Image: image,
//			Ports: oldrc.Spec.Template.Spec.Containers[0].Ports,
//		},
//	}

//	var newrc = &models.ReplicationController{
//		TypeMeta: models.TypeMeta{
//			Kind:       "ReplicationController",
//			APIVersion: "v1",
//		},
//		ObjectMeta: models.ObjectMeta{
//			Name:   newversion,
//			Labels: map[string]string{"name": service},
//		},
//		Spec: models.ReplicationControllerSpec{
//			Replicas: oldrc.Spec.Replicas,
//			Selector: map[string]string{"version": newversion},
//			Template: &models.PodTemplateSpec{
//				ObjectMeta: models.ObjectMeta{
//					Labels: map[string]string{"name": service, "version": newversion},
//				},
//				Spec: models.PodSpec{
//					Containers: containers,
//				},
//			},
//		},
//	}

//	body, _ = json.Marshal(newrc)
//	status, result := lib.Sendapi("POST", models.KubernetesIp, "8080", "v1", []string{"namespaces", namespace, "replicationcontrollers"}, body)
//	responsebody, _ := json.Marshal(result)
//	if status != 201 {
//		a.Ctx.ResponseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
//		a.Ctx.ResponseWriter.WriteHeader(status)
//		log.Fprintln(a.Ctx.ResponseWriter, string(responsebody))
//		return
//	}
//	//
//	var re = map[string]interface{}{}
//	re["create new rc"] = result
//	oldrc.Spec.Replicas = 0
//	body, _ = json.Marshal(oldrc)
//	log.Println(string(body))
//	_, result = lib.Sendapi("PUT", models.KubernetesIp, "8080", "v1", []string{"namespaces", namespace, "replicationcontrollers", oldrc.ObjectMeta.Name}, body)
//	re["close old pod"] = result
//	//time.Sleep(5 * time.Second)

//	_, result = lib.Sendapi("DELETE", models.KubernetesIp, "8080", "v1", []string{"namespaces", namespace, "replicationcontrollers", oldrc.ObjectMeta.Name}, []byte{})
//	re["delete old rc"] = result

//	_, exist := models.Appinfo[namespace]
//	if !exist {
//		models.Appinfo[namespace] = models.NamespaceInfo{}
//	}
//	_, exist = models.Appinfo[namespace][service]
//	if !exist {
//		models.Appinfo[namespace][service] = &models.AppMetaInfo{
//			Name:     service,
//			Replicas: newrc.Spec.Replicas,
//			Status:   1,
//		}
//	}
//	a.Data["json"] = re
//	a.ServeJson()
//}
