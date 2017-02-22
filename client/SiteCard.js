const React = require('react')
const h = require('react-hyperscript')
const R = require('recharts')
const DragSource = require('react-dnd').DragSource
const DropTarget = require('react-dnd').DropTarget
const Link = require('react-router-dom').Link

const graphql = require('./graphql')

const SiteCard = React.createClass({
  getInitialState () {
    return {
      site: {}
    }
  },

  q: `
    query siteOverview($code: String!) {
      site(code: $code) {
        name
        code
        days(last:7) {
          day
          s
          v
        }
      }
    }
  `,

  query () {
    graphql.query(this.q, {code: this.props.code})
    .then(r => this.setState(r))
    .catch(console.log.bind(console))
  },

  componentDidMount () {
    this.query()
  },

  componentWillReceiveProps (nextProps) {
    if (nextProps.code !== this.props.code) this.query()
  },

  render () {
    return this.props.connectDragSource(this.props.connectDropTarget(
      this.props.isDragging
      ? h('.card.site.empty', '　')
      : h('.card.site', {id: this.props.code}, [
        h('.card-content', [
          h('h4.title.is-4', this.state.site.name),
          h('h6.subtitle.is-6', [
            h(Link, {to: `/sites/${this.state.site.code}`}, this.state.site.code)
          ])
        ]),
        h('.card-image', [
          h('figure.image', [
            h(R.ResponsiveContainer, {height: 200, width: '100%'}, [
              h(R.ComposedChart, {data: this.state.site.days}, [
                h(R.Bar, {
                  dataKey: 's',
                  fill: '#8884d8',
                  isAnimationActive: false
                }),
                h(R.Line, {
                  dataKey: 'v',
                  stroke: '#82ca9d',
                  dot: false,
                  isAnimationActive: false
                })
              ])
            ])
          ])
        ])
      ])
    ))
  }
})

module.exports = DropTarget('site', {
  hover: function (props, monitor, component) {
    let dragIndex = monitor.getItem().index
    let hoverIndex = props.index

    // don't replace items with themselves
    if (dragIndex === hoverIndex) return

    props.moveSite(dragIndex, hoverIndex)

    // hack (from https://github.com/react-dnd/react-dnd/blob/5442d317f600d9760018f722c2968f6df0c2375c/examples/04%20Sortable/Simple/Card.js#L62-L66):
    monitor.getItem().index = hoverIndex
  },

  drop: function (props, monitor, component) {
    props.saveSiteOrder()
  }
}, function collect (connect) {
  return {
    connectDropTarget: connect.dropTarget()
  }
})(DragSource('site', {
  beginDrag: function (props) {
    return {code: props.code, index: props.index}
  }
}, function collect (connect, monitor) {
  return {
    connectDragSource: connect.dragSource(),
    isDragging: monitor.isDragging()
  }
})(SiteCard))
