const React = require('react')
const h = require('react-hyperscript')
const DragDropContext = require('react-dnd').DragDropContext
const TouchBackend = require('react-dnd-touch-backend')
const HTML5Backend = require('react-dnd-html5-backend')

const graphql = require('./graphql')
const SiteCard = require('./SiteCard')
const NewSiteCard = require('./NewSiteCard')

const Dashboard = React.createClass({
  getInitialState () {
    return {
      me: {
        sites: []
      }
    }
  },

  q: `
    query {
      me {
        sites {
          code
        }
      }
    }
  `,

  query () {
    graphql.query(this.q)
    .then(r => this.setState(r))
    .catch(console.log.bind(console))
  },

  componentDidMount () {
    this.query()
  },

  componentWillReceiveProps (nextProps) {},

  render () {
    return (
      h('.columns.is-multiline.is-mobile', this.state.me.sites.map((site, i) =>
        h('.column.is-one-quarter-desktop.is-one-third-tablet.is-half-mobile', {
          key: site.code
        }, [
          h(SiteCard, {
            code: site.code,
            index: i,
            moveSite: this.moveSite,
            saveSiteOrder: this.saveSiteOrder
          })
        ])
      ).concat(
        h('.column.is-one-quarter-desktop.is-one-third-tablet.is-half-mobile', {
          key: '_new'
        }, [
          h(NewSiteCard, {onNewSiteCreated: this.query})
        ])
      ))
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
  }
})

let backend = 'ontouchstart' in window || navigator.msMaxTouchPoints
  ? TouchBackend
  : HTML5Backend

module.exports = DragDropContext(backend)(Dashboard)
