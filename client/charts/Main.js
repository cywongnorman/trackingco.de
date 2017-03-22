const React = require('react')
const h = require('react-hyperscript')
const R = require('recharts')

const formatdate = require('../helpers').formatdate
const mergeColours = require('../helpers').mergeColours

module.exports = React.createClass({
  shouldComponentUpdate (nextProps, nextState) {
    return (this.props.site.days[0].day !== nextProps.site.days[0].day) ||
      (this.props.site.code !== nextProps.site.code) ||
      (this.props.site.days.length !== nextProps.site.days.length)
  },

  render () {
    let colours = mergeColours(this.props.colours)

    return (
      h(R.ResponsiveContainer, {height: 300, width: '100%'}, [
        h(R.ComposedChart, {data: this.props.site.days}, [
          h(R.XAxis, {dataKey: 'day', hide: true}),
          h(R.YAxis, {
            scale: 'linear',
            domain: [0, this.props.dataMax],
            orientation: 'right'
          }),
          h(R.Tooltip, {
            isAnimationActive: false,
            content: Tooltip
          }),
          h(R.Bar, {
            dataKey: 's',
            fill: colours.bar1
          }),
          h(R.Line, {
            dataKey: 'v',
            stroke: colours.line1,
            type: 'monotone',
            strokeWidth: 1
          })
        ])
      ])
    )
  }
})


const Tooltip = function (props) {
  return (
    h('div.custom-tooltip', [
      h('p.recharts-tooltip-label', formatdate(props.label)),
      h('ul.recharts-tooltip-item-list', props.payload.reverse().map(item =>
        h('li.recharts-tooltip-item', {style: {color: item.color}}, [
          h('span.recharts-tooltip-item-name', names[item.name]),
          h('span.recharts-tooltip-item-separator', ' : '),
          h('span.recharts-tooltip-item-value', item.value)
        ])
      ))
    ])
  )
}

const names = {
  s: 'unique sessions',
  v: 'all pageviews'
}
