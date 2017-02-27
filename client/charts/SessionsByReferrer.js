const React = require('react')
const h = require('react-hyperscript')
const R = require('recharts')

const colours = require('../helpers').colours

module.exports = React.createClass({
  shouldComponentUpdate (nextProps, nextState) {
    return (this.props.site.days[0].day !== nextProps.site.days[0].day) ||
      (this.props.site.code !== nextProps.site.code) ||
      (this.props.site.days.length !== nextProps.site.days.length)
  },

  render () {
    return (
      h(R.ResponsiveContainer, {height: 200, width: '100%'}, [
        h(R.BarChart, {
          data: this.props.site.sessionsbyreferrer.map(group =>
            group.scores.map(score => ({
              referrer: group.referrer,
              score: score
            }))
          ).reduce((a, b) => a.concat(b), [])
        }, [
          h(R.XAxis, {dataKey: 'referrer', hide: true}),
          h(R.Bar, {
            dataKey: 'score',
            fill: colours.bar2
          }),
          h(R.Tooltip, {
            isAnimationActive: false,
            content: Tooltip
          })
        ])
      ])
    )
  }
})


const Tooltip = function (props) {
  return props.payload.length
  ? (
    h('div.custom-tooltip', [
      h('ul.recharts-tooltip-item-list', [
        h('li.recharts-tooltip-props.payload[0]', {style: {color: props.payload[0].color}}, [
          h('span.recharts-tooltip-props.payload[0]-name', props.label),
          h('span.recharts-tooltip-props.payload[0]-separator', ' : '),
          h('span.recharts-tooltip-props.payload[0]-value', props.payload[0].value)
        ])
      ])
    ])
  )
  : h('div')
}
