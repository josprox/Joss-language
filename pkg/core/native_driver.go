package core

import (
	"fmt"
	"unsafe"
)

const maxNativeDriverResponse = 128 << 20

func callLoadedNativeDriver(driver *NativeDriverDefinition, method, argsJSON string) (string, error) {
	if driver == nil || driver.Call == nil {
		return "", fmt.Errorf("driver no cargado")
	}
	driver.Mu.Lock()
	defer driver.Mu.Unlock()
	pointer := driver.Call(method, argsJSON)
	if pointer == nil {
		return "", fmt.Errorf("joss_driver_call devolvio NULL")
	}
	if driver.Free != nil {
		defer driver.Free(pointer)
	}
	data := make([]byte, 0, 256)
	for index := 0; index < maxNativeDriverResponse; index++ {
		value := *(*byte)(unsafe.Add(unsafe.Pointer(pointer), index))
		if value == 0 {
			return string(data), nil
		}
		data = append(data, value)
	}
	return "", fmt.Errorf("respuesta excede %d MiB o no termina en NUL", maxNativeDriverResponse>>20)
}
