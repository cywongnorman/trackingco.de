const React = require('react')
const render = require('react-dom').render

import Main from './Main'

render(
  React.createElement(Main),
  document.getElementById('main')
)
