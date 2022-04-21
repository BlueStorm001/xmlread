/*
 * Copyright (c) 2021 BlueStorm
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFINGEMENT IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

// Package XmlReader xml fast parser
//
package XmlReader

import (
	"bytes"
	"github.com/BlueStorm001/bufferPool"
)

type XReader struct {
	bufferPool *bufferPool.BufferPool
	bufferMax  int //最大缓冲字节数
}

type XDecoder struct {
	bus   *bufferPool.ByteBuffer //原始记录
	len   int                    //bus长度
	index int                    //Read位置
	isc   bool
	b     byte //当前Read byte
	//buffer   bytes.Buffer //存储的缓冲节点
	buffer      []byte //存储的缓冲节点
	bufferMax   int    //最大缓冲字节数
	bufferIndex int    //缓冲位置
	reader      *XReader
}

type XToken struct {
	Name         string
	Attr         map[string]string
	Data         string
	StartElement bool
	EndElement   bool
	IsText       bool
	Finish       bool
}

// New 可在全局声明XReader
// max1:buffer池数量，给予最大并发量数值，默认500。
// max2:xml节点最大长度，默认500。例如：<root id="a"> 长度是13。
func New(max ...int) *XReader {
	var m1 = 500
	if len(max) > 0 {
		m1 = max[0]
	}
	var m2 = 500
	if len(max) > 1 {
		m2 = max[1]
	}
	return &XReader{
		bufferPool: bufferPool.New(m1),
		bufferMax:  m2,
	}
}

// Load 载入xml文档
func (r *XReader) Load(input []byte) *XDecoder {
	d := &XDecoder{}
	d.bus = r.bufferPool.Get()
	d.bus.Write(input)
	d.len = d.bus.Len()
	d.index = 0
	d.bufferMax = r.bufferMax
	d.buffer = make([]byte, d.bufferMax)
	d.bufferIndex = -1
	d.reader = r
	return d
}

func NewDefault(input string) *XDecoder {
	return Create(input, 500)
}

func Create(input string, bufferMax int) *XDecoder {
	return CreateBytes([]byte(input), bufferMax)
}

var byteBuffer = bufferPool.NewDefault()

func CreateBytes(input []byte, bufferMax int) *XDecoder {
	d := &XDecoder{}
	d.bus = byteBuffer.Get()
	d.bus.Write(input)
	d.len = d.bus.Len()
	d.index = 0
	d.bufferMax = bufferMax
	d.buffer = make([]byte, d.bufferMax)
	d.bufferIndex = -1
	return d
}

// Text 获取文本
func (d *XDecoder) Text() string {
	var b bytes.Buffer
	for i := d.index; i < d.len; i++ {
		if d.pred(i-1) == '/' && d.pred(i) == '>' {
			return ""
		}
		if d.pred(i) == '<' && d.pred(i+1) == '/' {
			d.index = i
			return b.String()
		}
		//fmt.Println(d.pred(i-1), d.pred(i), d.pred(i+1), d.pred(i+2), d.pred(i+3))
		if d.get(i) == '.' {
			return ""
		}
		b.WriteByte(d.b)
		if d.pred(i+1) == '<' && d.pred(i+2) == '/' {
			d.index = i
			return b.String()
		}
	}
	return ""
}

// Read 循环内读取xml
func (d *XDecoder) Read() XToken {
	var token = XToken{}
	for i := d.index; i < d.len; i++ {
		el := d.get(i)
		if el == '.' {
			return d.complete()
		}
		//只查找<
		if el == '<' {
			//获取ElementName
			for {
				i++
				el1 := d.get(i)
				if el1 == '.' {
					return d.complete()
				}
				//结束
				if el1 == '/' {
					for {
						i++
						el2 := d.get(i)
						if el2 == '.' {
							return d.complete()
						}
						if el2 == '_' {
							continue
						}
						if el2 == '@' {
							d.append()
						} else {
							token.Name = d.str()
							token.StartElement = false
							token.EndElement = true
							return d.next(i, token)
						}
					}
				}
				if el1 == '@' {
					d.append()
				} else {
					token.Name = d.str()
					token.StartElement = true
					token.EndElement = false
					break
				}
			}
			//结束本次
			if d.end() {
				return d.next(i, token)
			}
			//开始获取属性
			var k string
			for {
				i++
				e := d.get(i)
				if e == '.' {
					return d.complete()
				}
				if e == '_' {
					continue
				}
				if d.end() {
					return d.next(i, token)
				}
				if e == '@' {
					d.append()
				} else {
					if k == "" {
						k = d.str()
					}
				}
				//获得了KEY 再获取属性value
				if e == '=' && k != "" {
					do := 0
					//获取属性value
					for {
						i++
						if d.get(i) == '"' {
							do++
							if do == 1 {
								continue
							}
						}
						if do == 1 {
							d.append()
						} else if do == 2 {
							if token.Attr == nil {
								token.Attr = make(map[string]string)
							}
							token.Attr[k] = d.str()
							k = ""
							break
						}
					}
				}
			}
		}
	}
	return d.complete()
}

func (d *XDecoder) pred(i int) byte {
	if i >= d.len {
		return '.' //结束
	}
	return match(d.bus.B[i])
}

func (d *XDecoder) get(i int) byte {
	if i >= d.len {
		return '.' //结束
	}
	d.b = d.bus.B[i]
	return match(d.b)
}

func match(b byte) byte {
	switch b {
	case 60:
		return '<'
	case 47:
		return '/'
	case 62:
		return '>'
	case 32: // 空格
		return '_'
	case 39, 34: //单双引
		return '"'
	case 61:
		return '='
	default:
		return '@'
	}
}

func (d *XDecoder) end() bool {
	switch d.b {
	case '/':
		return true
	case '>':
		return true
	}
	return false
}

func (d *XDecoder) str() string {
	if d.bufferIndex < 0 {
		return ""
	}
	//切片
	buffer := d.buffer[0 : d.bufferIndex+1]
	//初始化
	d.bufferIndex = -1
	return string(buffer)
}

func (d *XDecoder) append() {
	//d.buffer = append(d.buffer, d.b)
	if d.bufferIndex >= d.bufferMax-1 {
		return
	}
	d.bufferIndex++
	d.buffer[d.bufferIndex] = d.b
}

func (d *XDecoder) next(i int, token XToken) XToken {
	d.index = i + 1
	return token
}

func (d *XDecoder) clear() {
	if d.reader == nil {
		byteBuffer.Put(d.bus)
	} else {
		d.reader.bufferPool.Put(d.bus)
	}
	d.len = 0
	d.b = 0
	d.index = 0
	d.isc = false
	d.buffer = nil
}

func (d *XDecoder) complete() XToken {
	var token = XToken{}
	token.Finish = true
	d.clear()
	return token
}
