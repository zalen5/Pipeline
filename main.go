package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ==========================
// Настройки
// ==========================
const (
	BufferSize    = 5               // размер кольцевого буфера
	FlushInterval = 5 * time.Second // интервал сброса буфера
)

// ==========================
// Кольцевой буфер
// ==========================
type CircularBuffer struct {
	data []int
	size int
	head int
	full bool
}

func NewCircularBuffer(size int) *CircularBuffer {
	return &CircularBuffer{
		data: make([]int, size),
		size: size,
	}
}

func (b *CircularBuffer) Add(value int) {
	b.data[b.head] = value
	b.head = (b.head + 1) % b.size
	if b.head == 0 {
		b.full = true
	}
}

func (b *CircularBuffer) Flush() []int {
	var output []int
	if b.full {
		output = append(output, b.data...)
	} else {
		output = append(output, b.data[:b.head]...)
	}
	b.data = make([]int, b.size)
	b.head = 0
	b.full = false
	return output
}

// ==========================
// Стадии конвейера
// ==========================

// 1. Фильтр отрицательных чисел
func filterNegative(input <-chan int) <-chan int {
	output := make(chan int)
	go func() {
		for val := range input {
			if val >= 0 {
				output <- val
			}
		}
		close(output)
	}()
	return output
}

// 2. Фильтр чисел, не кратных 3, и исключение нуля
func filterNotMultipleOf3(input <-chan int) <-chan int {
	output := make(chan int)
	go func() {
		for val := range input {
			if val != 0 && val%3 == 0 {
				output <- val
			}
		}
		close(output)
	}()
	return output
}

// 3. Буферизация с периодическим сбросом
func bufferStage(input <-chan int) <-chan []int {
	output := make(chan []int)
	buffer := NewCircularBuffer(BufferSize)

	go func() {
		ticker := time.NewTicker(FlushInterval)
		defer ticker.Stop()
		defer close(output)

		for {
			select {
			case val, ok := <-input:
				if !ok {
					// канал закрыт — сбрасываем буфер
					data := buffer.Flush()
					if len(data) > 0 {
						output <- data
					}
					return
				}
				buffer.Add(val)

			case <-ticker.C:
				data := buffer.Flush()
				if len(data) > 0 {
					output <- data
				}
			}
		}
	}()
	return output
}

// ==========================
// Источник данных (консоль)
// ==========================
func inputSource() <-chan int {
	output := make(chan int)
	go func() {
		defer close(output)
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Println("Введите целые числа (пустая строка — завершение):")

		for {
			fmt.Print("> ")
			if !scanner.Scan() {
				break
			}
			text := strings.TrimSpace(scanner.Text())
			if text == "" {
				break
			}
			val, err := strconv.Atoi(text)
			if err != nil {
				fmt.Println("Ошибка: введите целое число.")
				continue
			}
			output <- val
		}
	}()
	return output
}

// ==========================
// Потребитель данных
// ==========================
func consumer(input <-chan []int) {
	for batch := range input {
		fmt.Println("Получены данные:", batch)
	}
	fmt.Println("Конвейер завершён.")
}

// ==========================
// Основная функция
// ==========================
func main() {
	source := inputSource()
	stage1 := filterNegative(source)
	stage2 := filterNotMultipleOf3(stage1)
	stage3 := bufferStage(stage2)
	consumer(stage3)
}
