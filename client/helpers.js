const randomColor = require('randomcolor')
const color = require('color')
const levenshtein = require('leven')
const url = require('url')

const graphql = require('./graphql')

module.exports.coloursfragment = graphql.createFragment(`
  fragment on Colours {
    bar1
    line1
    background
  }
`)

module.exports.mergeColours = function () {
  var colours = defaultColours
  for (let i = 0; i < arguments.length; i++) {
    let colourSet = arguments[i]

    for (let field in colourSet) {
      if (colourSet[field]) {
        colours[field] = colourSet[field]
      }
    }
  }

  return colours
}

const defaultColours = {
  bar1: '#4791AE',
  line1: '#EA8676',
  background: '#FDEECD'
}

const months = require('months')
module.exports.formatdate = function formatdate (d) {
  if (d) {
    let month = months.abbr[parseInt(d.slice(4, 6)) - 1]
    return d.slice(6) + '/' + month + '/' + d.slice(0, 4)
  }
}

var colourCache = {}
module.exports.referrerColour = function referrerColour (referrer) {
  referrer = url.parse(referrer).host || referrer // handle "<direct>" case

  if (colourCache[referrer]) return colourCache[referrer]

  var smallerdistance = 15
  var nearest = ''
  for (var cachedReferrer in colourCache) {
    let distance = levenshtein(referrer, cachedReferrer)
    if (distance < nearest) {
      smallerdistance = distance
      nearest = cachedReferrer
    }
    if (smallerdistance < 3) break
  }

  if (smallerdistance > 7) {
    colourCache[referrer] = randomColor()
  } else {
    colourCache[referrer] = near(colourCache[nearest], smallerdistance)
  }

  return colourCache[referrer]
}

function near (hex, distance) {
  var [h, s, v] = color(hex).hsv().color
  h = h / 360
  s = s / 100
  v = v / 100

  let dist = distance / 10

  h += Math.random() / 30 + dist
  s = s > 0.5 ? s - (Math.random() / 10 + dist) : s + (Math.random() / 10 + dist)
  v = v > 0.5 ? v - (Math.random() / 10 + dist) : v + (Math.random() / 10 + dist)

  return color({
    h: mirror(h) * 360,
    s: mirror(s) * 100,
    v: mirror(v) * 100
  }).hex()
}

function mirror (value) {
  return value > 1
  ? value < 0
    ? (value + 1000) % 1
    : 1 - (value % 1)
  : value
}

module.exports.title = name => name ? `${name} | trackingco.de` : 'trackingco.de'
