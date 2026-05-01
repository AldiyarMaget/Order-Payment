import React, { useState, useEffect } from 'react'

class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error) {
    return { hasError: true };
  }

  componentDidCatch(error, errorInfo) {
    console.error("ErrorBoundary caught an error", error, errorInfo);
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="alert alert-error">
          Something went wrong while rendering the order list.
        </div>
      );
    }
    return this.props.children; 
  }
}

const API_BASE_URL = '/api';

function App() {
  const [orders, setOrders] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)
  
  const [formData, setFormData] = useState({
    customer_id: '',
    item_name: '',
    amount: ''
  })
  const [formStatus, setFormStatus] = useState(null) // {type: 'success'|'error', msg: ''}

  const fetchOrders = async () => {
    setLoading(true)
    setError(null)
    try {
      // Fetch broadly just to show everything for Midterm
      const res = await fetch(`${API_BASE_URL}/orders?min_amount=0&max_amount=9999999`)
      if (!res.ok) {
        throw new Error('Failed to fetch orders')
      }
      const data = await res.json()
      setOrders(data || [])
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchOrders()
    // Auto-refresh every 5 seconds for "streaming-like" monitoring feeling
    const intervalId = setInterval(fetchOrders, 5000)
    return () => clearInterval(intervalId)
  }, [])

  const handleInputChange = (e) => {
    const { name, value } = e.target
    setFormData(prev => ({...prev, [name]: value}))
  }

  const handleCreateOrder = async (e) => {
    e.preventDefault()
    setFormStatus(null)

    if (!formData.customer_id || !formData.item_name || !formData.amount) {
      setFormStatus({type: 'error', msg: 'All fields are required.'})
      return
    }

    try {
      const payload = {
        customer_id: formData.customer_id,
        item_name: formData.item_name,
        amount: parseInt(formData.amount, 10)
      }

      const res = await fetch(`${API_BASE_URL}/orders`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Idempotency-Key': crypto.randomUUID()
        },
        body: JSON.stringify(payload)
      })

      if (!res.ok) {
        const errData = await res.json()
        throw new Error(errData.error || 'Failed to create order')
      }

      setFormStatus({type: 'success', msg: 'Order created successfully!'})
      setFormData({customer_id: '', item_name: '', amount: ''})
      fetchOrders() // Refresh immediately
    } catch (err) {
      setFormStatus({type: 'error', msg: err.message})
    }
  }

  const getStatusClass = (status) => {
    const s = (status || '').toLowerCase()
    if (s.includes('paid')) return 'bg-paid'
    if (s.includes('fail') || s.includes('cancel')) return 'bg-failed'
    return 'bg-pending'
  }

  return (
    <div className="app-container">
      <header className="header">
        <h1>SRE Control Center</h1>
        <p>Order and Payment Delivery Monitoring</p>
      </header>

      <div className="grid">
        <div className="glass-panel form-panel">
          <div className="panel-title">Create Order</div>
          
          {formStatus && (
            <div className={`alert alert-${formStatus.type}`}>
              {formStatus.msg}
            </div>
          )}

          <form onSubmit={handleCreateOrder}>
            <div className="form-group">
              <label>Customer ID (UUID)</label>
              <input 
                type="text" 
                name="customer_id"
                value={formData.customer_id}
                onChange={handleInputChange}
                className="form-control" 
                placeholder="e.g. 550e8400-e29b..."
              />
            </div>
            <div className="form-group">
              <label>Item Name</label>
              <input 
                type="text" 
                name="item_name"
                value={formData.item_name}
                onChange={handleInputChange}
                className="form-control" 
                placeholder="Product or Service"
              />
            </div>
            <div className="form-group">
              <label>Amount</label>
              <input 
                type="number" 
                name="amount"
                value={formData.amount}
                onChange={handleInputChange}
                className="form-control" 
                placeholder="1000"
                min="1"
              />
            </div>
            <button type="submit" className="btn">Deploy Order</button>
          </form>
        </div>

        <div className="glass-panel list-panel">
          <div className="panel-title">
            Recent Orders
            <button onClick={fetchOrders} className="btn-secondary" title="Refresh">
              {loading ? 'Refreshing...' : 'Refresh'}
            </button>
          </div>

          {error && <div className="alert alert-error">{error}</div>}

          <ErrorBoundary>
            <div className="table-container">
              <table>
                <thead>
                  <tr>
                    <th>ID</th>
                    <th>Customer</th>
                    <th>Item</th>
                    <th>Amount</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>
                  {(!orders || orders.length === 0) && !loading ? (
                    <tr>
                      <td colSpan="5" style={{textAlign: "center", fontStyle: "italic", opacity: 0.7}}>No orders found.</td>
                    </tr>
                  ) : (
                    Array.isArray(orders) && orders.map(order => (
                      <tr key={order.id || Math.random()}>
                        <td style={{fontFamily: "monospace", fontSize: "0.85rem"}}>{order.id?.substring(0,8) || 'N/A'}...</td>
                        <td>{order.customer_id || 'N/A'}</td>
                        <td>{order.item_name || 'N/A'}</td>
                        <td>{order.amount ?? 'N/A'}</td>
                        <td>
                          <span className={`status-badge ${getStatusClass(order.status)}`}>
                            {order.status || 'Unknown'}
                          </span>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </ErrorBoundary>
        </div>
      </div>
    </div>
  )
}

export default App
