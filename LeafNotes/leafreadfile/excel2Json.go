package main

import (
	"github.com/Luxurioust/excelize"
	"fmt"
	"os"
	"io/ioutil"
	"path"
	"regexp"
	"strings"
	"path/filepath"
	"time"
)

//默认sheet的下标，从1开始
var ActiveSheet = 1
var Directory = "jsonConfigs"
//匹配模式段
var RegexpPattern = "^#.*"
//生成的目录
var JsonDir = "Json"

const (
	IntType = iota //Int类型
	FloatType //Float类型
	ArrType //数组类型
	Other //其它类型
)
//读取表格
func readExcel(filaname string){
	xlsx, err := excelize.OpenFile(filaname)
	if err != nil {
		fmt.Printf("The filename [%v] [%v] \n",filaname,err)
		return
	}
	//设置默认的sheet
	//获取默认的sheet的Name
	xlsx.SetActiveSheet(ActiveSheet)
	sheetName := xlsx.GetSheetName(xlsx.GetActiveSheetIndex())
	//字段名
	fieldName := []string{}
	//get the rows of sheetName xlsx
	rows := xlsx.GetRows(sheetName)

	//分离目录和文件
	josnDir,jsonF := path.Split(filaname)
	jsonF = strings.TrimSuffix(jsonF,".xlsx")+".json"
	jsonF = path.Join(josnDir,JsonDir,jsonF)
	//创建json文件
	jsonFile,err := os.Create(jsonF)
	if err != nil {
		fmt.Printf("create json file [%v] \n",err)
		return
	}
	defer jsonFile.Close()
	jsonFile.WriteString("[\n")
	for rowindex, row := range rows {
		switch rowindex {
		case 0:
			continue
		case 1:
			continue
		case 2:

		default:
			jsonFile.WriteString("\t{\n")
		}

		for k, colCell := range row {
			switch rowindex {
			case 1:
				if k == 0 {
					isReg,errReg := regexp.MatchString(RegexpPattern,colCell)
					if errReg != nil {
						fmt.Printf("match string pattern[%v] string[%v]",RegexpPattern,colCell)
						return
					}
					if !isReg {
						fmt.Printf("the first row must be comment \n")
						return
					}
				}
			case 2:
				if colCell == "" ||colCell == " "{
					continue
				}
				fieldName = append(fieldName,colCell)
			default:
				if k>= len(fieldName){
					continue
				}
				strType,strResult := ReflectValue(colCell)
				switch strType {
				case IntType:
					strResult = fmt.Sprintf("\t\"%s\":%s",fieldName[k],colCell)
				case FloatType:
					strResult = fmt.Sprintf("\t\"%s\":%s",fieldName[k],colCell)
				case ArrType:
					strResult = fmt.Sprintf("\t\"%s\":%s",fieldName[k],colCell)
				case Other:
					strResult = fmt.Sprintf("\t\"%s\":\"%s\"",fieldName[k],colCell)
				default:
					strResult = fmt.Sprintf("\t\"%s\":\"%s\"",fieldName[k],colCell)
				}
				if k+1 == len(fieldName) {
				}else{
					strResult += ",\n"
				}

				jsonFile.WriteString(strResult)
			}

		}
		switch rowindex {
		case 0:
		case 1:
		case 2:
		case len(rows)-1:
			jsonFile.WriteString("\n\t}\n")
		default:
			jsonFile.WriteString("\n\t},\n")
		}
	}
	jsonFile.WriteString("]")

}

//convert excel to json
func excel2json(name string)error{
	fileInfos,err :=ioutil.ReadDir(name)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	for _,v := range fileInfos {
		if v.IsDir() {
			continue
		}
		fileName := v.Name()
		ext := path.Ext(fileName)
		if ext != ".xlsx"{
			continue
		}
		fmt.Println("当前处理表格文件为：",fileName)
		readExcel(fileName)
		fmt.Println(fileName,"处理完成")
	}
	return nil
}

func createDir(name string){
	flag := RemoveAll(path.Join(name,JsonDir))
	if !flag {
		fmt.Println("Remove the configs has some problems")
		return
	}
	fmt.Println("创建Json文件夹")
	err := os.Mkdir(JsonDir,os.ModeDir)
	if err != nil {
		fmt.Println("创建json文件夹失败",err)
		return
	}
	excel2json(name)
}


//check the directory and remove the all files
func RemoveAll(name string)(bool){
	_,err := os.Stat(name)
	if err == nil {
		errRe := os.RemoveAll(name)
		if errRe != nil {
			fmt.Println(errRe)
			return false
		}
		return true
	}
	if os.IsExist(err) {
		errRe := os.RemoveAll(name)
		if errRe != nil {
			fmt.Println(errRe)
			return false
		}
		return true
	}
	return true
}

//check the value type
func ReflectValue(str string)(int,string){
	okArr,_ := regexp.MatchString("^([\\[])+",str)
	if okArr{
		return ArrType,str
	}


	result := strings.Split(str,".")
	if len(result) != 2 && len(result) != 1 {
		return Other,str
	}

	if len(result) == 1 {
		ok2,_ := regexp.MatchString("^([0-9])+",result[0])
		if!ok2{
			return Other,str
		}
		return IntType,str
	}

	if len(result) == 2 {
		if ok2,_ := regexp.MatchString("^[0-9]+",result[0]);!ok2{
			return Other,str
		}

		if ok,_:=regexp.MatchString("^[0-9]+",result[1]);!ok {
			return Other,str
		}
	}

	return FloatType,str

}

func ReflectArr(str string){
	//temp := str

	str = strings.Replace(str,"[","",-1)
	str = strings.Replace(str,"]","",-1)
	first := strings.Split(str,",")[0]
	tempFirst := strings.Split(first,".")
	if len(tempFirst) !=1 &&len(tempFirst)!= 2 {


		return
	}
	if  len(tempFirst) ==1{
		left := tempFirst[0]
		if ok,_ := regexp.MatchString("^[0-9]+",left);ok{
			return
		}else{
			return
		}
	}
	if  len(tempFirst) ==2{
		left := tempFirst[0]
		if ok,_ := regexp.MatchString("^[0-9]+",left);ok{
			right := tempFirst[1]
			if ok2,_ := regexp.MatchString("^[0-9]+",right);ok2 {
				return
			}else{

			}
		}else{
			return
		}
	}
}

func main(){
	nowDir :=getCurrentPath()
	fmt.Println("")
	printNoBug()
	fmt.Println("=======================================")
	fmt.Println("开始转换")
	fmt.Println("当前地址：",nowDir)
	time.Sleep(4*time.Second)
	createDir(nowDir)
	fmt.Println("=======================================")
	fmt.Println("全部处理完成")
	time.Sleep(2*time.Second)
}

func getCurrentPath() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Println(err)
	}
	return dir
}

func printNoBug(){
	fmt.Println("                            _ooOoo_")
	fmt.Println("                           o8888888o")
	fmt.Println("                           88  .  88")
	fmt.Println("                           (| -_- |)")
	fmt.Println("                          0\\    =  /0")
	fmt.Println("                        ____/`---'\\____")
	fmt.Println("                      .'  \\\\|     |//  `.")
	fmt.Println("                     /  \\\\|||  :  |||//  \\")
	fmt.Println("                    /  _||||| -:- |||||-  \\")
	fmt.Println("                    |   | \\\\\\  -  /// |   |")
	fmt.Println("                    | \\_|  ''\\---/''  |   |")
	fmt.Println("                    \\  .-\\__  `-`  ___/-. /")
	fmt.Println("                  ___`. .'  /--.--\\  `. . __")
	fmt.Println("               .\"\" '<  `.___\\_<|>_/___.'  >'\"\".")
	fmt.Println("              | | :  `- \\`.;`\\ _ /`;.`/ - ` : | |")
	fmt.Println("              \\  \\ `-.   \\_ __\\ /__ _/   .-` /  /")
	fmt.Println("         ======`-.____`-.___\\_____/___.-`____.-'======")
	fmt.Println("                            `=---='")
	fmt.Println("        ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^")
	fmt.Println("                      Buddha Bless, No Bug !")
}