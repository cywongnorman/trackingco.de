const React = require('react')
const render = require('react-dom').render
const h = require('react-hyperscript')
const Lokka = require('lokka').Lokka
const Transport = require('lokka-transport-http').Transport

const graphql = new Lokka({ transport: new Transport('/_graphql') })

const Card = React.createClass({
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
        days(last:14) {
          sessions
          pageviews
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
    this.setState({code: nextProps.code}, this.query)
  },

  render () {
    return h('.card', [
      h('.card-header', [
        h('h4.card-title', this.state.site.name),
        h('h6.card-meta', this.state.site.code)
      ]),
      h('.card-image', [

      ])
    ])
  }
})

render(
  React.createElement(Card, { code: 'ciz4n47q800010z090rwz1rxn' }),
  document.getElementById('main')
)
