const React = require('react')
const h = require('react-hyperscript')
const R = require('recharts')
const n = require('format-number')({})

const formatmonth = require('../helpers').formatmonth
const mergeColours = require('../helpers').mergeColours

module.exports = React.createClass({
  shouldComponentUpdate (nextProps, nextState) {
    return (this.props.site.months[0].month !== nextProps.site.months[0].months) ||
      (this.props.site.code !== nextProps.site.code) ||
      (this.props.site.months.length !== nextProps.site.months.length)
  },

  render () {
    let colours = mergeColours(this.props.colours)

    return (
      h(R.ResponsiveContainer, {height: 300, width: '100%'}, [
        h(R.ComposedChart, {data: this.props.site.months}, [
          h(R.XAxis, {dataKey: 'month', hide: true}),
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
      h('p.recharts-tooltip-label', formatmonth(props.label)),
      h('ul.recharts-tooltip-item-list', props.payload.reverse().map(item =>
        h('li.recharts-tooltip-item', {style: {color: item.color}}, [
          h('span.recharts-tooltip-item-name', names[item.name]),
          h('span.recharts-tooltip-item-separator', ' : '),
          h('span.recharts-tooltip-item-value', n(item.value))
        ])
      ))
    ])
  )
}

const names = {
  s: 'unique sessions',
  v: 'all pageviews'
}
