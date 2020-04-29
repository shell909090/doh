### Makefile --- 

## Author: shell@dsk
## Version: $Id: Makefile,v 0.0 2020/04/29 14:02:48 shell Exp $
## Keywords: 
## X-URL: 
LEVEL=NOTICE

clean:
	rm -rf bin pkg gopath debuild *.log

build:
	mkdir -p gopath/src/github.com/shell909090/
	ln -s "$$PWD" gopath/src/github.com/shell909090/doh
	mkdir -p bin
	GOPATH="$$PWD/gopath":"$$GOPATH" go build -o bin/goproxy github.com/shell909090/doh/doh
	rm -rf gopath


### Makefile ends here
