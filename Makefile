build:
	go build -o dist/mp4dovi

install:
	rm -f ~/bin/mp4dovi
	cp -f dist/mp4dovi ~/bin/
