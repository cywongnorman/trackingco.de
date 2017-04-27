const React = require('react')
const h = require('react-hyperscript')
const DragSource = require('react-dnd').DragSource
const DropTarget = require('react-dnd').DropTarget

const log = require('./log')
const graphql = require('./graphql')
const coloursfragment = require('./helpers').coloursfragment
const charts = {
  Card: require('./charts/Card')
}

const SiteCard = React.createClass({
  getInitialState () {
    return {
      site: null,
      me: null
    }
  },

  sitef: graphql.createFragment(`
fragment on Site {
  name
  code
  shareURL
  days {
    day
    s
    v
  }
}
  `),

  query () {
    graphql.query(`
query c($code: String!) {
  site(code: $code, last: 7) {
    ...${this.sitef}
  }

  me {
    colours { ...${coloursfragment} }
  }
}
    `, {code: this.props.code})
    .then(r => this.setState(r))
    .catch(log.error)
  },

  componentDidMount () {
    this.query()
  },

  componentWillReceiveProps (nextProps) {
    if (nextProps.code !== this.props.code) this.query()
  },

  render () {
    if (!this.state.site) {
      return h('div')
    }

    return this.props.connectDragSource(this.props.connectDropTarget(
      this.props.isDragging
      ? h('.card.site.empty', '　')
      : h('.card.site', {id: this.props.code}, [
        h('.card-header', [
          h('p.card-header-title', [
            h('a', {href: `/sites/${this.state.site.code}`}, this.state.site.name)
          ]),
          this.state.site.shareURL
          ? (
            h('p.card-header-icon', {title: 'this site is publicly shared'}, [
              h('a.icon', [
                h('i.fa.fa-share-alt')
              ])
            ])
          )
          : ''
        ]),
        h('.card-image', [
          h('i.fa.fa-square-o.placeholder'),
          h('figure.image', [
            h(charts.Card, {
              site: this.state.site,
              colours: this.state.me.colours
            })
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
