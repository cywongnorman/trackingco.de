const React = require('react')
const render = require('react-dom').render

const Dashboard = require('./Dashboard')

render(
  React.createElement(Dashboard),
  document.getElementById('main')
)
