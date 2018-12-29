const React = require('react')
const render = require('react-dom').render

const log = require('./log')
const Main = require('./Main')

render(
  React.createElement(Main),
  document.getElementById('main')
)

// log anything that is in location.hash
let tolog = location.hash.slice(1).split('=')
if (tolog[0] === 'log' && tolog[1]) {
  log.info(tolog[1])
  location.hash = ''
}
