/** @format */

const randomColor = require('randomcolor')
const color = require('color')
const levenshtein = require('leven')
const url = require('url')
const golangFormat = require('nice-time')
const months = require('months')

export const MONTH = 'MONTH'
export const DAY = 'DAY'
export const DATEFORMAT = '20060102'
export const MONTHFORMAT = '200601'

export const defaultColours = {
  bar1: '#4791AE',
  line1: '#EA8676',
  background: '#FDEECD'
}

export function mergeColours() {
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

var colourCache = {}
export function referrerColour(referrer) {
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

function near(hex, distance) {
  var [h, s, v] = color(hex).hsv().color
  h = h / 360
  s = s / 100
  v = v / 100

  let dist = distance / 10

  h += Math.random() / 30 + dist
  s =
    s > 0.5 ? s - (Math.random() / 10 + dist) : s + (Math.random() / 10 + dist)
  v =
    v > 0.5 ? v - (Math.random() / 10 + dist) : v + (Math.random() / 10 + dist)

  return color({
    h: mirror(h) * 360,
    s: mirror(s) * 100,
    v: mirror(v) * 100
  }).hex()
}

function mirror(value) {
  return value > 1 ? (value < 0 ? (value + 1000) % 1 : 1 - (value % 1)) : value
}

export const title = name =>
  name ? `${name} | trackingco.de` : 'trackingco.de'

function makefill(kind) {
  var format
  var previous
  var next
  var frombeggining
  var key

  if (kind === MONTH) {
    format = MONTHFORMAT
    previous = curr => {
      let prev = new Date(curr)
      prev.setMonth(prev.getMonth() - 1)
      return prev
    }
    next = curr => {
      let next = new Date(curr)
      next.setMonth(next.getMonth() + 1)
      return next
    }
    frombeggining = (start, offset) => {
      let current = new Date(start)
      current.setMonth(current.getMonth() - offset)
      return current
    }
    key = 'month'
  } else if (kind === DAY) {
    format = DATEFORMAT
    previous = curr => {
      let prev = new Date(curr)
      prev.setDate(prev.getDate() - 1)
      return prev
    }
    frombeggining = (start, offset) => {
      let current = new Date(start)
      current.setDate(current.getDate() - offset)
      return current
    }
    next = curr => {
      let next = new Date(curr)
      next.setDate(next.getDate() + 1)
      return next
    }
    key = 'day'
  }

  return function(periods, offset, start) {
    // fill in missing periods (days/months) with zeroes
    var allperiods = new Array(offset)

    start = start || new Date()

    var prev = previous(start)
    var current = frombeggining(start, offset)

    var currentpos = 0
    var rowpos = 0

    while (current < prev) {
      let currentDate = golangFormat(format, current)

      if (periods[rowpos][key] === currentDate) {
        allperiods[currentpos] = periods[rowpos]
        rowpos++
      } else {
        allperiods[currentpos] = {[key]: currentDate}
      }

      current = next(current)
      currentpos++
    }

    return allperiods
  }
}

export const fillmonths = makefill(MONTH)
export const filldays = makefill(DAY)

export function encodedate(date) {
  return golangFormat(DATEFORMAT, date)
}
export function encodemonth(date) {
  return golangFormat(MONTHFORMAT, date)
}

export function formatdate(d) {
  if (d) {
    let month = months.abbr[parseInt(d.slice(4, 6)) - 1]
    return d.slice(6) + '/' + month + '/' + d.slice(0, 4)
  }
}
export function formatmonth(d) {
  if (d) {
    let month = months.abbr[parseInt(d.slice(4, 6)) - 1]
    return month + '/' + d.slice(0, 4)
  }
}

export function mapToEntryList(map) {
  var list = []
  for (let addr in map) {
    let amount = map[addr]
    list.push({a: addr, c: amount})
  }
  return list.sort((a, b) => b.c - a.c)
}
