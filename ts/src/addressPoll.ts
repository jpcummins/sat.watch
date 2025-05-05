// Create a polling component to hydrate address balances via AlpineJS

declare const Alpine: any;

interface AddressStatus {
  id: string;
  utxoCount: number;
  balance: number;
}

// Register Alpine component on alpine:init for CSP compliance
document.addEventListener('alpine:init', () => {
	Alpine.data('addressPoll', () => ({
		pending: [] as string[],

		init(): void {
			// Use setTimeout to ensure DOM is fully rendered, especially table rows
			setTimeout(() => {
				// Refine selector: target pending cells first
				const pendingCells = document.querySelectorAll<HTMLElement>('td[data-pending="true"]');
				this.pending = Array.from(pendingCells)
					.map(td => td.closest('tr')?.dataset.addressId!) // Get ID from parent row
					.filter(Boolean) // Remove any undefined/null IDs if closest('tr') failed
					.filter((v, i, a) => a.indexOf(v) === i); // Unique IDs

				this.poll();
				setInterval(() => this.poll(), 5000);
			}, 0); // Defer execution slightly
		},

		async poll(): Promise<void> {
			if (!this.pending.length) {
				return;
			}

			const ids = encodeURIComponent(this.pending.join(','));
			const res = await fetch(`/app/addresses/status?ids=${ids}`, {
				headers: { 'Accept': 'application/json' }
			});
			if (!res.ok) return;

			const updates = (await res.json()) as AddressStatus[];
			// Add check to ensure updates is an array before calling forEach
			if (!Array.isArray(updates)) {
				return; // Stop processing this poll cycle
			}

			updates.forEach(u => {
				const row = document.querySelector<HTMLElement>(`[data-address-id="${u.id}"]`);
				if (!row) return;
				const utxoTd = row.querySelector<HTMLElement>('.utxo-count-cell')!;
				const balTd  = row.querySelector<HTMLElement>('.balance-cell')!;

				utxoTd.textContent = u.utxoCount.toString();
				balTd.textContent  = u.balance.toString();

				utxoTd.dataset.pending = 'false';
				balTd.dataset.pending  = 'false';
			});

			// remove finished IDs
			this.pending = this.pending.filter(id => !updates.some(u => u.id === id));
		}
	}));
}); 