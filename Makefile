all: static/bundle.js static/style.css

prod: static/bundle.min.js static/style.min.css
	mv static/bundle.min.js static/bundle.js
	mv static/style.min.css static/style.css

watch:
	find client | entr make

static/bundle.js: $(shell find client/)
	godotenv -f .env ./node_modules/.bin/browserifyinc client/app.js -dv --outfile static/bundle.js

static/bundle.min.js:  $(shell find client/)
	./node_modules/.bin/browserify client/app.js -t babelify -g [ envify --NODE_ENV production ] -g uglifyify | ./node_modules/.bin/uglifyjs --compress --mangle > static/bundle.min.js

static/style.css: client/style.styl
	./node_modules/.bin/stylus < client/style.styl > static/style.css

static/style.min.css: client/style.styl
	./node_modules/.bin/stylus -c < client/style.styl > static/style.min.css
