const React = require('react')
const h = require('react-hyperscript')

const graphql = require('./graphql')
const SiteCard = require('./SiteCard')
const NewSiteCard = require('./NewSiteCard')

module.exports = React.createClass({
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
    return h('.columns.is-multiline.is-mobile', this.state.me.sites.map(site =>
      h('.column.is-one-quarter-desktop.is-one-third-tablet.is-half-mobile', {
        key: site.code
      }, [
        h(SiteCard, {code: site.code})
      ])
    ).concat(
      h('.column.is-one-quarter-desktop.is-one-third-tablet.is-half-mobile', {
        key: '_new'
      }, [
        h(NewSiteCard, {onNewSiteCreated: this.query})
      ])
    ))
  }
})
