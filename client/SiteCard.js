const React = require('react')
const h = require('react-hyperscript')
const DragSource = require('react-dnd').DragSource
const DropTarget = require('react-dnd').DropTarget
const Link = require('react-router-dom').Link

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
      me: null,
      editing: false,
      deleting: false,
      editingName: ''
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
            this.state.editing
            ? h('span', {
              contentEditable: true,
              onInput: e => {
                this.setState({editingName: e.target.innerHTML.trim()})
              },
              ref: el => { el && el.focus() },
              dangerouslySetInnerHTML: {
                __html: this.state.site.name
              }
            })
            : h(Link, {to: `/sites/${this.state.site.code}`}, this.state.site.name)
          ]),
          this.state.site.shareURL
          ? (
            h('p.card-header-icon', {title: 'this site is shared'}, [
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
        ]),
        h('.card-footer', this.state.deleting
          ? [
            h('a.card-footer-item.danger', {onClick: this.confirmDelete}, 'really delete?'),
            h('a.card-footer-item', {onClick: this.cancelDelete}, 'cancel')
          ]
          : this.state.editing
            ? [
              h('a.card-footer-item.save', {onClick: this.confirmEdit}, 'save'),
              h('a.card-footer-item', {onClick: this.cancelEdit}, 'cancel')
            ]
            : [
              h(Link, {className: 'card-footer-item', to: `/sites/${this.props.code}`}, 'view'),
              h('a.card-footer-item', {onClick: () => { this.setState({editing: true}) }}, 'rename'),
              h('a.card-footer-item', {onClick: () => { this.setState({deleting: true}) }}, 'delete')
            ]
        )
      ])
    ))
  },

  confirmEdit () {
    if (!this.state.editingName) {
      this.setState({editing: false})
      return
    }

    graphql.mutate(`
      ($name: String!, $code: String!) {
        renameSite(name: $name, code: $code) {
          ...${this.sitef}
        }
      }
    `, {name: this.state.editingName, code: this.state.site.code})
    .then(r => {
      this.setState({editing: false, site: r.renameSite})
    })
    .catch(log.error)
  },

  cancelEdit () {
    this.setState({editing: false})
  },

  confirmDelete () {
    this.setState({deleting: false})
    graphql.mutate(`
($code: String!) {
  deleteSite(code: $code) {
    ok, error
  }
}
    `, {code: this.state.site.code})
    .then(r => {
      if (!r.deleteSite.ok) {
        log.error('failed to delete site:', r.deleteSite.error)
        this.setState({deleting: false})
        return
      }
      log.info(`${this.props.site.name} was deleted.`)
      this.props.iWasDeleted()
    })
    .catch(log.error)
  },

  cancelDelete () {
    this.setState({deleting: false})
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
