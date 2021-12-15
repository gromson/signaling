package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type Data struct {
	PI   float64
	Rest []uint8
}

func main() {
	b := []byte{0x18, 0x2d, 0x44, 0x54, 0xfb, 0x21, 0x09, 0x40, 0xff, 0x01, 0x02, 0x03, 0xbe, 0xef}
	r := bytes.NewReader(b)

	data := Data{
		PI: 0,
		Rest: make([]uint8, len(b)-8),
	}

	if err := binary.Read(r, binary.LittleEndian, &data); err != nil {
		fmt.Println("binary.Read failed:", err)
	}

	fmt.Printf("%#v\n", data)
}
