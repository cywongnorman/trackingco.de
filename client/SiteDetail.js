const React = require('react')
const h = require('react-hyperscript')
const R = require('recharts')

const graphql = require('./graphql')

module.exports = React.createClass({
  render () {
    return h('div')
    //         h(R.XAxis, {dataKey: 'day', hide: true}),
    //         h(R.Tooltip, {
    //           labelFormatter: d => d.slice(6) + '/' + d.slice(4, 6) + '/' + d.slice(0, 4),
    //           isAnimationActive: false
    //         }),
  }
})

