"use client"

import { CartesianGrid, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';

export default function BalanceGraph() {
  // Sample data - replace with your actual data
  const data = [
    { date: '4/30', amount: 20000 },
    { date: '5/15', amount: 40000 },
    // ... more data points
  ];

  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-xl font-semibold text-gray-800">Activity</h2>
        <span className="text-2xl font-bold text-green-600">$4,456.53</span>
        <a href="/transfers" className="text-blue-600 hover:text-blue-800 transition-colors">Transfers</a>
      </div>
      <ResponsiveContainer width="100%" height={300}>
        <LineChart data={data}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="date" />
          <YAxis />
          <Tooltip />
          <Line type="monotone" dataKey="amount" stroke="#8884d8" strokeWidth={2} />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
