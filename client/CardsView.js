const React = require('react')
const h = require('react-hyperscript')
const DragDropContext = require('react-dnd').DragDropContext
const TouchBackend = require('react-dnd-touch-backend')
const HTML5Backend = require('react-dnd-html5-backend')
const DocumentTitle = require('react-document-title')
const BodyStyle = require('body-style')

const graphql = require('./graphql')
const SiteCard = require('./SiteCard')
const NewSiteCard = require('./NewSiteCard')
const mergeColours = require('./helpers').mergeColours
const onLoggedStateChange = require('./auth').onLoggedStateChange

const Dashboard = React.createClass({
  getInitialState () {
    return {
      sites: null,
      me: null
    }
  },

  query () {
    graphql.query(`
query {
  me {
    sites {
      code
    }
    colours { background }
  }
}
    `)
    .then(r => this.setState(r))
    .catch(console.log.bind(console))
  },

  componentDidMount () {
    onLoggedStateChange(isLogged => {
      if (isLogged) {
        this.query()
      }
    })
  },

  render () {
    if (!this.state.me) {
      return h('div')
    }

    let backgroundColor = mergeColours(this.state.me.colours).background

    return (
      h(DocumentTitle, {title: 'sites'}, [
        h(BodyStyle, {style: {backgroundColor}}, [
          h('.columns.is-multiline.is-mobile', this.state.me.sites.map((site, i) =>
            h('.column.is-2-widescreen.is-3-desktop.is-4-tablet.is-6-mobile', {
              key: site.code
            }, [
              h(SiteCard, {
                code: site.code,
                index: i,
                moveSite: this.moveSite,
                saveSiteOrder: this.saveSiteOrder,
                iWasDeleted: this.removeSiteFromScreen.bind(this, site.code)
              })
            ])
          ).concat(
            h('.column.is-2-widescreen.is-3-desktop.is-4-tablet.is-6-mobile', {
              key: '_new'
            }, [
              h(NewSiteCard, {onNewSiteCreated: this.query})
            ])
          ))
        ])
      ])
    )
  },

  moveSite (dragIndex, hoverIndex) {
    this.setState(st => {
      let moving = st.me.sites[dragIndex]
      st.me.sites.splice(dragIndex, 1)
      st.me.sites.splice(hoverIndex, 0, moving)
      return st
    })
  },

  saveSiteOrder () {
    let order = this.state.me.sites.map(s => s.code)

    graphql.mutate(`
      ($order: [String]!) {
        changeSiteOrder(order: $order) {
          ok
        }
      }
    `, {order: order})
    .then(r => {
      console.log(r)
    })
    .catch(e => {
      console.log(e.stack)
    })
  },

  removeSiteFromScreen (code) {
    this.setState(st => {
      st.me.sites = st.me.sites.filter(s => s.code !== code)
      return st
    })
  }
})

let backend = 'ontouchstart' in window || navigator.msMaxTouchPoints
  ? TouchBackend
  : HTML5Backend

module.exports = DragDropContext(backend)(Dashboard)
