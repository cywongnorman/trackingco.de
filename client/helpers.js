module.exports.colours = {
  bar1: '#8884d8',
  bar2: '#a7533e',
  line1: '#82ca9d'
}

module.exports.formatdate = function (d) {
  if (d) {
    let month = months.abbr[parseInt(d.slice(4, 6)) - 1]
    return d.slice(6) + '/' + month + '/' + d.slice(0, 4)
  }
}

const months = require('months')
