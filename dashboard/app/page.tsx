"use client";

import BalanceGraph from "@/components/BalanceGraph";
import Header from "@/components/Header";
import RecentGrants from "@/components/RecentGrants";
import Sidebar from "@/components/Sidebar";
import TopNonprofits from "@/components/TopNonprofits";
import { useEffect, useState } from "react";

export default function Dashboard() {
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);

  useEffect(() => {
    const handleResize = () => {
      setIsSidebarCollapsed(window.innerWidth < 1024); // Collapse sidebar on screens smaller than 1024px
    };

    handleResize(); // Initial check
    window.addEventListener('resize', handleResize);

    return () => window.removeEventListener('resize', handleResize);
  }, []);

  return (
    <div className="flex flex-col min-h-screen">
      <div className="fixed top-0 left-0 right-0 z-10">
        <Header />
      </div>
      <div className="flex flex-1 pt-16"> {/* Add padding-top to account for fixed header */}
        <Sidebar isCollapsed={isSidebarCollapsed} />
        <main className={`flex-1 p-6 overflow-y-auto transition-all duration-300`}> {/* Add overflow-y-auto to enable scrolling */}
          <div className="flex flex-col lg:flex-row lg:space-x-6 mb-6">
            <div className="w-full lg:w-2/3 mb-6 lg:mb-0">
              <BalanceGraph />
            </div>
            <div className="w-full lg:w-1/3">
              <TopNonprofits />
            </div>
          </div>
          <RecentGrants />
        </main>
      </div>
    </div>
  );
}
