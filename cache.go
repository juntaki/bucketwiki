package main

import (
	"errors"
	"log"

	"github.com/juntaki/transparent"
	"github.com/juntaki/transparent/custom"
	"github.com/juntaki/transparent/lru"
)

func (w *Wikidata) initializeMarkdownCache() error {
	getfunc := func(key interface{}) (interface{}, error) {
		titleHash, _ := key.(string)
		log.Println("Get from source", titleHash)
		page, err := w.loadMarkdown(titleHash, "")
		if err != nil {
			return nil, err
		}
		return page, nil
	}

	addfunc := func(key interface{}, value interface{}) (err error) {
		pageData, _ := value.(*pageData)
		log.Println("Set source", key)
		err = w.saveMarkdown(pageData)
		return err
	}

	removefunc := func(key interface{}) (err error) {
		titleHash, _ := key.(string)
		err = w.deleteMarkdown(titleHash)
		return err
	}
	storage, err := custom.NewStorage(getfunc, addfunc, removefunc)
	if err != nil {
		return err
	}
	source, err := transparent.NewLayerSource(storage)
	if err != nil {
		return err
	}
	cache, _ := lru.NewCache(10, 10)
	w.pageCache = transparent.NewStack()
	w.pageCache.Stack(source)
	w.pageCache.Stack(cache)
	w.pageCache.Start()

	return nil
}

func (w *Wikidata) loadMarkdownAsync(titleHash string) (*pageData, error) {
	value, err := w.pageCache.Get(titleHash)
	if err != nil {
		return nil, err
	}
	pagedata, ok := value.(*pageData)
	if !ok {
		return nil, errors.New("Failed to get page data")
	}
	return pagedata, nil
}

func (w *Wikidata) saveMarkdownAsync(titleHash string, pageData *pageData) error {
	return w.pageCache.Set(titleHash, pageData)
}

func (w *Wikidata) deleteMarkdownAsync(titleHash string) error {
	return w.pageCache.Remove(titleHash)
}

// for UserData
func (w *Wikidata) initializeUserCache() error {
	getfunc := func(key interface{}) (interface{}, error) {
		name, _ := key.(string)
		log.Println("Get from source", name)
		user, err := w.loadUser(name)
		if err != nil {
			return nil, err
		}
		return user, nil
	}

	addfunc := func(key interface{}, value interface{}) (err error) {
		userData, _ := value.(*userData)
		log.Println("Set source", key)
		err = w.saveUser(userData)
		return err
	}

	removefunc := func(key interface{}) (err error) {
		name, _ := key.(string)
		err = w.deleteUser(name)
		return err
	}
	storage, err := custom.NewStorage(getfunc, addfunc, removefunc)
	if err != nil {
		return err
	}
	source, err := transparent.NewLayerSource(storage)
	if err != nil {
		return err
	}
	cache, _ := lru.NewCache(10, 10)
	w.userCache = transparent.NewStack()
	w.userCache.Stack(source)
	w.userCache.Stack(cache)
	w.userCache.Start()

	return nil
}

func (w *Wikidata) loadUserAsync(name string) (*userData, error) {
	value, err := w.userCache.Get(name)
	if err != nil {
		return nil, err
	}
	userdata, ok := value.(*userData)
	if !ok {
		return nil, errors.New("Failed to get user data")
	}
	return userdata, nil
}

func (w *Wikidata) saveUserAsync(name string, userData *userData) error {
	return w.userCache.Set(name, userData)
}

func (w *Wikidata) deleteUserAsync(name string) error {
	return w.userCache.Remove(name)
}

// for File
func (w *Wikidata) initializeFileCache() error {
	getfunc := func(key interface{}) (interface{}, error) {
		fileDataKey, _ := key.(fileDataKey)
		log.Println("Get from source", fileDataKey)
		file, err := w.loadFile(fileDataKey)
		if err != nil {
			return nil, err
		}
		return file, nil
	}

	addfunc := func(key interface{}, value interface{}) (err error) {
		fileData, _ := value.(*fileData)
		log.Println("Set source", key)
		err = w.saveFile(fileData)
		return err
	}

	removefunc := func(key interface{}) (err error) {
		fileDataKey, _ := key.(fileDataKey)
		err = w.deleteFile(fileDataKey)
		return err
	}
	storage, err := custom.NewStorage(getfunc, addfunc, removefunc)
	if err != nil {
		return err
	}
	source, err := transparent.NewLayerSource(storage)
	if err != nil {
		return err
	}
	cache, _ := lru.NewCache(10, 10)
	w.fileCache = transparent.NewStack()
	w.fileCache.Stack(source)
	w.fileCache.Stack(cache)
	w.fileCache.Start()

	return nil
}

func (w *Wikidata) loadFileAsync(key fileDataKey) (*fileData, error) {
	value, err := w.fileCache.Get(key)
	if err != nil {
		return nil, err
	}
	filedata, ok := value.(*fileData)
	if !ok {
		return nil, errors.New("Failed to get file data")
	}
	return filedata, nil
}

func (w *Wikidata) saveFileAsync(key fileDataKey, fileData *fileData) error {
	return w.fileCache.Set(key, fileData)
}

func (w *Wikidata) deleteFileAsync(key fileDataKey) error {
	return w.fileCache.Remove(key)
}
