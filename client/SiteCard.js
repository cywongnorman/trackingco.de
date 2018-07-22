const React = require('react')
const h = require('react-hyperscript')
const DragSource = require('react-dnd').DragSource
const DropTarget = require('react-dnd').DropTarget

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

  render () {
    return this.props.connectDragSource(this.props.connectDropTarget(
      this.props.isDragging
        ? h('.card.site.empty', '　')
        : h('.card.site', {id: this.props.site.code}, [
          h('.card-header', [
            h('p.card-header-title', [
              h('a', {href: `/sites/${this.props.site.code}`}, this.props.site.name)
            ]),
            this.props.site.shareURL
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
                site: this.props.site,
                colours: {}
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
    return {code: props.site.code, index: props.index}
  }
}, function collect (connect, monitor) {
  return {
    connectDragSource: connect.dragSource(),
    isDragging: monitor.isDragging()
  }
})(SiteCard))
