# xml read


``` golang
import (
    "fmt"
    "github.com/BlueStorm001/xmlread"
)
```
``` golang
var str = `<?xml version="1.0" encoding="utf-8" ?>
            <a>
                <a1 id="a1id">a321</a1>
                <bb/>
                <b>
                    <a:Code>UO</a:Code>
                    <cs>
                        <c>
                            <name>"&/'>'</name>
                            <age>18</age>
                        </c>
                        <c>
                            <name><>>&&''""</name>
                            <age>19</age>
                        </c>
                    </cs>
                </b>
            </a>`
            
var xml = xmlread.New()
```
``` golang
func main() {
    var r = xml.Load(b)
    //循环读取
    for {
        read := r.Read()
        if read.Finish {
            break
        }
        if read.Name == "" {
            continue
        }
        if read.StartElement && read.Name == "a1" {
            fmt.Println(read.Name, r.Text(), read.Attr["id"])
        }
        if read.StartElement && read.Name == "a:Code" {
            fmt.Println(read.Name, r.Text())
        }
        if read.StartElement && read.Name == "name" {
            fmt.Println(read.Name, r.Text())
        }
        if read.StartElement && read.Name == "age" {
            fmt.Println(read.Name, r.Text())
        }
    }
}
```