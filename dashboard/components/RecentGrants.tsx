
export default function RecentGrants() {

  // Sample data - replace with actual data from your API
  const grants = [
    { id: 1, amount: 7500.00, recipient: "Susan G. Komen", donor: "John Doe", status: 'Pending', transactions: 'Not reviewed', date: 'September 20, 2024, 5:18 PM' },
    { id: 2, amount: 501.00, recipient: "American Cancer Society", donor: "Jane Smith", status: 'Pending', transactions: 'Not reviewed', date: 'September 20, 2024, 5:18 PM' },
    { id: 3, amount: 1252.00, recipient: "Central Park Conservatory", donor: "Alice Johnson", status: 'Pending', transactions: 'Not reviewed', date: 'September 20, 2024, 11:09 AM' },
    { id: 4, amount: 465.60, recipient: "Memorial Sloan Kettering", donor: "Bob Williams", status: 'Received', transactions: 'Not reviewed', date: 'September 20, 2024, 11:33 PM' },
    // Add more grant objects as needed
  ];

  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-xl font-bold text-gray-800">Recent Grants</h2>
        <span className="bg-blue-100 text-blue-800 text-sm font-medium mr-2 px-2.5 py-0.5 rounded">441</span>
        <a href="/transfers" className="text-blue-600 hover:text-blue-800">View all</a>
      </div>
      <table className="w-full text-sm text-left text-gray-500">
        <thead className="text-xs text-gray-700 uppercase bg-gray-50">
          <tr>
            <th scope="col" className="px-6 py-3">Amount</th>
            <th scope="col" className="px-6 py-3">Recipient</th>
            <th scope="col" className="px-6 py-3">Donor</th>
            <th scope="col" className="px-6 py-3">Transactions</th>
            <th scope="col" className="px-6 py-3">Date</th>
          </tr>
        </thead>
        <tbody>
          {grants.map((grant) => (
            <tr key={grant.id} className="bg-white border-b hover:bg-gray-50">
              <td className="px-6 py-4 font-medium text-gray-900">${grant.amount.toFixed(2)}</td>
              <td className="px-6 py-4">
                <span className={`inline-block w-2 h-2 mr-2 rounded-full ${grant.status.toLowerCase() === 'pending' ? 'bg-yellow-400' : 'bg-green-400'}`}></span>
                {grant.recipient}
              </td>
              <td className="px-6 py-4">{grant.donor}</td>
              <td className="px-6 py-4">
                <button className="text-gray-400 hover:text-gray-500 mr-2">👁️</button>
                <button className="text-gray-400 hover:text-gray-500 mr-2">🔄</button>
                {grant.transactions}
              </td>
              <td className="px-6 py-4">{grant.date}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
