const React = require('react')
const h = require('react-hyperscript')
const R = require('recharts')
const color = require('color')

const colours = require('../helpers').colours

module.exports = React.createClass({
  render () {
    return (
      h(R.ResponsiveContainer, {height: 200, width: '100%'}, [
        h(R.ComposedChart, {data: this.props.site.days}, [
          h(R.Bar, {
            dataKey: 's',
            fill: color(colours.bar1).lighten(0.2).string(),
            isAnimationActive: false
          }),
          h(R.Line, {
            dataKey: 'v',
            stroke: color(colours.line1).lighten(0.2).string(),
            type: 'monotone',
            strokeWidth: 3,
            dot: false,
            isAnimationActive: false
          })
        ])
      ])
    )
  }
})
