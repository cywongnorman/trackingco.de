const React = require('react')
const render = require('react-dom').render

const Main = require('./Main')

render(
  React.createElement(Main),
  document.getElementById('main')
)
