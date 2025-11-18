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

func (b *CircularBuffer) Len() int {
	if b.full {
		return b.size
	}
	return b.head
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
		defer close(output)
		for val := range input {
			fmt.Printf("[Stage1: Фильтр отрицательных чисел] Получено значение: %d\n", val)
			if val >= 0 {
				fmt.Printf("[Stage1: Фильтр отрицательных чисел] → Пропущено (неотрицательное): %d\n", val)
				output <- val
			} else {
				fmt.Printf("[Stage1: Фильтр отрицательных чисел] ✗ Отфильтровано (отрицательное): %d\n", val)
			}
		}
		fmt.Println("[Stage1: Фильтр отрицательных чисел] Стадия завершена")
	}()
	return output
}

// 2. Фильтр чисел, не кратных 3, и исключение нуля
func filterNotMultipleOf3(input <-chan int) <-chan int {
	output := make(chan int)
	go func() {
		defer close(output)
		for val := range input {
			fmt.Printf("[Stage2: Фильтр чисел, не кратных 3] Получено значение: %d\n", val)
			if val != 0 && val%3 == 0 {
				fmt.Printf("[Stage2: Фильтр чисел, не кратных 3] → Пропущено (кратно 3): %d\n", val)
				output <- val
			} else {
				if val == 0 {
					fmt.Printf("[Stage2: Фильтр чисел, не кратных 3] ✗ Отфильтровано (ноль)\n")
				} else {
					fmt.Printf("[Stage2: Фильтр чисел, не кратных 3] ✗ Отфильтровано (не кратно 3): %d\n", val)
				}
			}
		}
		fmt.Println("[Stage2: Фильтр чисел, не кратных 3] Стадия завершена")
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

		fmt.Println("[Stage3: Буферизация] Стадия буферизации запущена")

		for {
			select {
			case val, ok := <-input:
				if !ok {
					// входной канал закрыт → финальный сброс
					fmt.Println("[Stage3: Буферизация] Входной канал закрыт → финальный сброс")
					data := buffer.Flush()
					if len(data) > 0 {
						fmt.Printf("[Stage3: Буферизация] → Финальный батч отправлен: %v\n", data)
						output <- data
					}
					fmt.Println("[Stage3: Буферизация] Стадия завершена")
					return
				}

				// обычное добавление
				buffer.Add(val)
				fmt.Printf("[Stage3: Буферизация] Добавлено: %d  (размер буфера: %d/%d)\n", val, buffer.Len(), BufferSize)

			case <-ticker.C:
				data := buffer.Flush()
				if len(data) > 0 {
					fmt.Println("[Stage3: Буферизация] Таймер → сброс буфера")
					fmt.Printf("[Stage3: Буферизация] → Отправлен батч: %v\n", data)
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
				fmt.Println("[Input] Ввод прерван (EOF)")
				break
			}
			text := strings.TrimSpace(scanner.Text())
			if text == "" {
				fmt.Println("[Input] Пустая строка — завершение ввода")
				break
			}

			val, err := strconv.Atoi(text)
			if err != nil {
				fmt.Printf("[Input] Ошибка ввода: %s — %v\n", text, err)
				continue
			}

			fmt.Printf("[Input] Принято и отправлено в пайплайн: %d\n", val)
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
		fmt.Printf("[Consumer] Получен батч: %v\n", batch)
		fmt.Println("Получены данные:", batch)
	}
	fmt.Println("[Consumer] Конвейер завершён.")
}

// ==========================
// Основная функция
// ==========================
func main() {
	fmt.Println("=== Пайплайн запущен с детальным логированием действий ===")
	source := inputSource()
	stage1 := filterNegative(source)
	stage2 := filterNotMultipleOf3(stage1)
	stage3 := bufferStage(stage2)
	consumer(stage3)
}
