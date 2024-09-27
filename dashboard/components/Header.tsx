
export default function Header() {
  return (
    <header className="flex justify-between items-center p-4 bg-white shadow-md">
      <div className="text-xl font-bold text-gray-800">Chai-Life Line dba ...</div>
      <nav className="flex items-center space-x-4">
        <button className="bg-blue-500 hover:bg-blue-600 text-white font-bold py-2 px-4 rounded">
          Move money
        </button>
        <button className="bg-gray-200 hover:bg-gray-300 rounded-full p-2">
          <span className="text-gray-600 font-bold">?</span>
        </button>
        <button className="bg-gray-200 hover:bg-gray-300 rounded-full p-2">
          <span className="text-gray-600">👤</span>
        </button>
      </nav>
    </header>
  );
}
