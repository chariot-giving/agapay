export default function TopNonprofits() {
  const nonprofits = [
    { id: 1, name: 'American Cancer Society', logo: 'acsf.png', ein: '13-1788491' },
    { id: 2, name: 'Susan G. Komen', logo: 'komen.png', ein: '75-1835298' },
    { id: 3, name: 'Central Park Conservatory', logo: 'cpc.png', ein: '13-3022855' },
    { id: 4, name: 'Memorial Sloan Kettering', logo: 'msk.png', ein: '13-1924236' },
    // Add more nonprofit objects as needed
  ];

  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-xl font-bold text-gray-800">Recipients</h2>
        <span className="bg-blue-100 text-blue-800 text-sm font-medium mr-2 px-2.5 py-0.5 rounded">9</span>
        <a href="/manage-connections" className="text-blue-600 hover:text-blue-800">Manage</a>
      </div>
      <ul className="space-y-4">
        {nonprofits.map((nonprofit) => (
          <li key={nonprofit.id} className="flex items-center space-x-3">
            <img
              src={`/images/${nonprofit.logo}`}
              alt={`${nonprofit.name} logo`}
              className="w-10 h-10 rounded-full"
            />
            <div className="flex flex-col">
              <span className="text-sm font-medium text-gray-700">{nonprofit.name}</span>
              <span className="text-xs text-gray-500">EIN: {nonprofit.ein}</span>
            </div>
          </li>
        ))}
      </ul>
      <div className="mt-4 text-right text-sm text-gray-500">
        <span>9 of 9</span>
      </div>
    </div>
  );
}
