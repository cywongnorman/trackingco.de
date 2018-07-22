all: bundle.js

bundle.js: $(shell ag -l --js --ignore bundle.js --ignore bundle.min.js)
	./node_modules/.bin/browserifyinc -vd app.js -o bundle.js

bundle.min.js: bundle.js
	cp bundle.js bundle.min.js

watch:
	ag -l --js --ignore bundle.js --ignore bundle.min.js --ignore browserify-cache.json --ignore node_modules | entr make

