const React = require('react')
const h = require('react-hyperscript')
const R = require('recharts')

const referrerColour = require('../helpers').referrerColour

module.exports = React.createClass({
  shouldComponentUpdate (nextProps, nextState) {
    if (nextProps.site !== this.props.site) {
      return true
    }
    return false
  },

  render () {
    return (
      h(R.ResponsiveContainer, {height: 200, width: '100%'}, [
        h(R.BarChart, {
          data: this.props.individualSessions,
          barGap: '3%'
        }, [
          h(R.XAxis, {dataKey: 'referrer', hide: true}),
          h(R.Bar, {
            dataKey: 'score',
            onClick: this.props.handleClick
          }, this.props.individualSessions.map((session, i) =>
            h(R.Cell, {key: i, fill: referrerColour(session.referrer)})
          )),
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

module.exports.sessionGroupsToIndividual = sessionGroupsToIndividual
function sessionGroupsToIndividual (sessiongroups) {
  return sessiongroups.map(group =>
    group.scores.map(score => ({
      referrer: group.referrer,
      score
    }))
  ).reduce((a, b) => a.concat(b), [])
}
