package paginator

import (
	"errors"
	"math"
	"reflect"
)

// DefaultMaxPerPage default number of records per page
const DefaultMaxPerPage = 10

var (
	// ErrNoPrevPage current page is first page
	ErrNoPrevPage = errors.New("no previous page")

	// ErrNoNextPage current page is last page
	ErrNoNextPage = errors.New("no next page")
)

type Page struct {
	// total pages
	TotalPages int `json:"totalPages,omitempty"`
	// total elements size
	TotalElements int64 `json:"totalElements,omitempty"`
	// elements size in current page
	NumberOfElements int `json:"numberOfElements,omitempty"`
	// elements size in single page
	Size int `json:"size,omitempty"`
	// current page number
	Number int `json:"number,omitempty"`
	// stores the current page results into data argument which must be a pointer to a slice.
	Content interface{} `json:"content,omitempty"`
}

type (
	// Adapter any adapter must implement this interface
	Adapter interface {
		Nums() (int64, error)
		Slice(offset, length int, data interface{}) error
	}

	// Paginator interface
	Paginator interface {
		SetPage(page int)
		Page() (int, error)
		Results(data interface{}) error
		PageResults(page int, data interface{}) (*Page, error)
		ToPageResults(data interface{}) (*Page, error)
		Nums() (int64, error)
		HasPages() (bool, error)
		HasNext() (bool, error)
		PrevPage() (int, error)
		NextPage() (int, error)
		HasPrev() (bool, error)
		PageNums() (int, error)
	}

	// Paginator structure
	paginator struct {
		adapter    Adapter
		maxPerPage int
		page       int
		nums       int64
	}
)

// New paginator constructor
func New(adapter Adapter, maxPerPage int) Paginator {
	if maxPerPage <= 0 {
		maxPerPage = DefaultMaxPerPage
	}

	return &paginator{
		adapter:    adapter,
		maxPerPage: maxPerPage,
		page:       1,
		nums:       -1,
	}
}

// SetPage set current page
func (p *paginator) SetPage(page int) {
	if page <= 0 {
		page = 1
	}

	p.page = page
}

// Page returns current page
func (p paginator) Page() (int, error) {
	pn, err := p.PageNums()
	if err != nil {
		return 0, err
	}

	if p.page > pn {
		return pn, nil
	}

	return p.page, nil
}

// Results stores the current page results into data argument which must be a pointer to a slice.
func (p paginator) Results(data interface{}) error {
	var offset int
	page, err := p.Page()
	if err != nil {
		return err
	}

	if page > 1 {
		offset = (page - 1) * p.maxPerPage
	}

	return p.adapter.Slice(offset, p.maxPerPage, data)
}

// PageResults stores the current page results into data argument which must be a pointer to a slice. then return to a page struct.
func (p paginator) PageResults(page int, data interface{}) (*Page, error) {
	p.SetPage(page)

	// Find current page data
	err := p.Results(data)
	if err != nil {
		return nil, err
	}

	// Return to page struct
	return p.ToPageResults(data)
}

// Results stores the current page results into data argument which must be a pointer to a slice.
func (p paginator) ToPageResults(data interface{}) (*Page, error) {
	nums, err := p.Nums()
	if err != nil {
		return nil, err
	}

	pageNums, err := p.PageNums()
	if err != nil {
		return nil, err
	}

	page, err := p.Page()
	if err != nil {
		return nil, err
	}

	// Identity slice type then fetch the size
	numberOfElements := 0
	switch reflect.TypeOf(data).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(data)
		numberOfElements = s.Len()
	}

	pageResult := &Page{
		TotalPages:       pageNums,
		TotalElements:    nums,
		Size:             p.maxPerPage,
		Number:           page,
		NumberOfElements: numberOfElements,
		Content:          data,
	}

	return pageResult, nil
}

// Nums returns the total number of records
func (p *paginator) Nums() (int64, error) {
	var err error
	if p.nums == -1 {
		p.nums, err = p.adapter.Nums()
		if err != nil {
			return 0, err
		}
	}

	return p.nums, nil
}

// HasPages returns true if there is more than one page
func (p paginator) HasPages() (bool, error) {
	n, err := p.Nums()
	if err != nil {
		return false, err
	}

	return n > int64(p.maxPerPage), nil
}

// HasNext returns true if current page is not the last page
func (p paginator) HasNext() (bool, error) {
	pn, err := p.PageNums()
	if err != nil {
		return false, err
	}

	page, err := p.Page()
	if err != nil {
		return false, err
	}

	return page < pn, nil
}

// PrevPage returns previous page number or ErrNoPrevPage if current page is first page
func (p paginator) PrevPage() (int, error) {
	hp, err := p.HasPrev()
	if err != nil {
		return 0, nil
	}

	if !hp {
		return 0, ErrNoPrevPage
	}

	page, err := p.Page()
	if err != nil {
		return 0, err
	}

	return page - 1, nil
}

// NextPage returns next page number or ErrNoNextPage if current page is last page
func (p paginator) NextPage() (int, error) {
	hn, err := p.HasNext()
	if err != nil {
		return 0, err
	}

	if !hn {
		return 0, ErrNoNextPage
	}

	page, err := p.Page()
	if err != nil {
		return 0, err
	}

	return page + 1, nil
}

// HasPrev returns true if current page is not the first page
func (p paginator) HasPrev() (bool, error) {
	page, err := p.Page()
	if err != nil {
		return false, err
	}

	return page > 1, nil
}

// PageNums returns the total number of pages
func (p paginator) PageNums() (int, error) {
	n, err := p.Nums()
	if err != nil {
		return 0, err
	}

	n = int64(math.Ceil(float64(n) / float64(p.maxPerPage)))
	if n == 0 {
		n = 1
	}

	return int(n), nil
}
