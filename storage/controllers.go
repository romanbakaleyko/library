package storage

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"path/filepath"
	"strconv"

	"github.com/twinj/uuid"
)

var (
	// ErrNotFound describe the state when the object is not found in the storage
	ErrNotFound = errors.New("can't find the book with given ID")
)

type library struct {
	storage string
	//storage io.ReadWriteCloser // Here you can put opened os.File object. After that you will be able to implement concurrent safe operations with file storage
	useSql bool
}

// NewLibrary constructor for library struct.
// Constructors are often used for initialize some data structures (map, slice, chan...)
// or when you need some data preparation
// or when you want to start some watchers (goroutines). In this case you also have to think about Close() method.
func NewLibrary(pathToStorage string, useSql bool) *library {
	return &library{
		storage: pathToStorage,
		useSql:  useSql,
	}
}

func (l *library) writeData(books Books) error {
	path, err := filepath.Abs(l.storage)
	if err != nil {
		return err
	}

	booksBytes, err := json.MarshalIndent(books, "", "    ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, booksBytes, 0644)
}

func (l *library) wantedIndex(id string, books Books) (int, error) {
	for index, book := range books {
		if id == book.ID {
			return index, nil
		}
	}
	return 0, ErrNotFound
}

//GetBooks returns all book objects
func (l *library) GetBooks() (Books, error) {
	var books Books

	if l.useSql {
		// Connection to the database
		db, err := InitDB()
		if err != nil {
			return nil, err
		}
		// Close connection database
		defer db.Close()
		// SELECT * FROM books
		if err = db.Find(&books).Error; err != nil {
			return nil, err
		}

		return books, nil
	}

	path, err := filepath.Abs(l.storage)
	if err != nil {
		return nil, err
	}

	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return books, json.Unmarshal(file, &books)
}

// CreateBook adds book object into db
func (l *library) CreateBook(book Book) error {
	err := errors.New("not all fields are populated")
	switch {
	case book.Genres == nil:
		return err
	case book.Pages == 0:
		return err
	case book.Price == 0:
		return err
	case book.Title == "":
		return err
	}

	book.ID = uuid.NewV4().String()
	if l.useSql {
		// Connection to the database
		db, err := InitDB()
		if err != nil {
			return err
		}
		// Close connection database
		defer db.Close()

		return db.Create(&book).Error
	}

	books, err := l.GetBooks()
	if err != nil {
		return err
	}

	books = append(books, book)
	return l.writeData(books)
}

// GetBook returns book object with specified id
func (l *library) GetBook(id string) (Book, error) {
	var b Book
	if l.useSql {
		// Connection to the database
		db, err := InitDB()
		if err != nil {
			return b, err
		}
		// Close connection database
		defer db.Close()

		if err = db.Where("id = ?", id).First(&b).Error; err != nil {
			return b, err
		}

		return b, nil
	}

	books, err := l.GetBooks()
	if err != nil {
		return b, err
	}

	for _, book := range books {
		if id == book.ID {
			return book, nil
		}
	}
	return b, ErrNotFound
}

// RemoveBook removes book object with specified id
func (l *library) RemoveBook(id string) error {
	if l.useSql {
		var book Book
		// Connection to the database
		db, err := InitDB()
		if err != nil {
			return err
		}
		// Close connection database
		defer db.Close()
		if err = db.Where("id = ?", id).First(&book).Error; err != nil {
			return err
		}

		if err = db.Delete(&book).Error; err != nil {
			return err
		}

		return nil
	}

	books, err := l.GetBooks()
	if err != nil {
		return err
	}

	index, err := l.wantedIndex(id, books)
	if err != nil {
		return err
	}
	books = append(books[:index], books[index+1:]...)
	return l.writeData(books)
}

// ChangeBook updates book object with specified id
func (l *library) ChangeBook(id string, changedBook Book) error {
	if l.useSql {
		var book Book
		// Connection to the database
		db, err := InitDB()
		if err != nil {
			return err
		}
		// Close connection database
		defer db.Close()
		if err = db.Where("id = ?", id).First(&book).Error; err != nil {
			return err
		}
		if err = db.Save(&changedBook).Error; err != nil {
			return err
		}
		return nil
	}

	books, err := l.GetBooks()
	if err != nil {
		return err
	}

	index, err := l.wantedIndex(id, books)
	if err != nil {
		return err
	}

	book := &books[index]
	book.Price = changedBook.Price
	book.Title = changedBook.Title
	book.Pages = changedBook.Pages
	book.Genres = changedBook.Genres
	err = l.writeData(books)
	return err
}

// PriceFilter returns filtered book objects
func (l *library) PriceFilter(filter BookFilter) (Books, error) {
	var wantedBooks Books

	if l.useSql {
		return wantedBooks, errors.New("NotImplemented")
	}
	if len(filter.Price) <= 1 {
		return nil, errors.New("Not valid data")
	}
	operator := string(filter.Price[0])
	if operator != "<" && operator != ">" {
		err := errors.New("unsupported operation")
		return nil, err
	}

	books, err := l.GetBooks()
	if err != nil {
		return nil, err
	}

	price, err := strconv.ParseFloat(filter.Price[1:], 64)
	if err != nil {
		return nil, err
	}

	for _, book := range books {
		if operator == ">" {
			if book.Price > price {
				wantedBooks = append(wantedBooks, book)
			}
		} else {
			if book.Price < price {
				wantedBooks = append(wantedBooks, book)
			}
		}
	}
	return wantedBooks, nil
}
