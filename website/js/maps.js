/**
 * Maps class for displaying maps on the website.
 * Dependencies: Leaflet.js, Luxon.js
 */
class Maps {
    // Default configuration
    #config = {}
    #participants = []
    #markers = []
    #firstDisplay = true
    #expire = null

    constructor(configuration) {
        this.#config = Object.assign(this.#config, configuration);
    }

    /**
     * Create an instance of Leaflet in the element identified by 'map' id.
     * By default, the map shows all the world
     * @returns {void}
     */
    display() {
        this.map = L.map('map')
        this.map.setView([0.0, 0.0], 3);

        L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
            maxZoom: 19, attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
        }).addTo(this.map)

    }

    /**
     * Displays the expiration date of the map
     * @returns {void}
     */
    async getExpiration() {
        const mapInfoResponse = await fetch(`/api/v1/map/${this.#config.publicId}`)
        if (mapInfoResponse.status !== 200) {
            console.log('error')
            return
        }
        const mapInfo = await mapInfoResponse.json()
        this.#expire = DateTime.fromSeconds(mapInfo.expire);
    }

    /**
     * Refreshes the duration of the map
     * @returns {void}
     */
    refreshTTL() {
        if (this.#expire === null) {
            return
        }
        document.getElementById('ttl').innerHTML =this.#expire.toRelative({round: true})
        document.getElementById('expire').title = this.#expire.toLocaleString(DateTime.DATETIME_SHORT)
    }

    /**
     * Displays the last position of the user on the map and refresh the local storage
     * @returns {void}
     */
    async displayLastPosition() {
        const positionsResponse = await fetch(`/api/v1/map/${this.#config.publicId}/positions`)
        if (positionsResponse.status !== 200) {
            console.log('error')
            return
        }
        this.#markers = []
        const positions = await positionsResponse.json()
        for (const position of positions.lastPositions) {
            const marker = L.marker([position.latitude, position.longitude]).addTo(this.map);
            marker.nickname = position.nickname;
            const since = DateTime.fromSeconds(position.timestamp);
            marker.bindTooltip(`${position.nickname} - ${since.toRelative({ round: false, style: 'short'})}`);
            this.#markers.push(marker)
            const currentParticipant = position.nickname;
            const storedParticipant = this.#participants.findIndex(participant => participant.nickname === currentParticipant)
            if (storedParticipant !== -1) {
                this.#participants[storedParticipant] = position
            } else {
                this.#participants.push(position);
            }
        }

        if (this.#firstDisplay) {
            this.#firstDisplay = false
            this.fitParticipants()
        }
    }

    /**
     * Displays the list of participants on the map
     * @returns {void}
     */
    displaysParticipants() {
        if (this.#participants.length === 0) {
            document.getElementById('participants').innerHTML = '<li>😭 No participants</li>'
            return
        }
        for (const participant of this.#participants) {
            if (document.getElementById(`${participant.nickname}`)) // if participant is already displayed do not display it again
            {
                continue
            }

            document.getElementById('participants').innerHTML += `<li id="${participant.nickname}"><a onclick="zoom('${participant.nickname}')" class="cursor-pointer hover:font-bold">${participant.nickname}</a></li>`
        }
    }

    /**
     * Zoom the map to display all the participants
     * @returns {void}
     */
    fitParticipants() {
        if (this.#markers.length === 0) {
            return
        }
        this.map.fitBounds(this.#markers.map(marker => marker.getLatLng()));
    }

    /**
     * Zoom the map to display a specific participant
     * @param {string} nickname
     * @returns {void}
     */
    fitParticipant(nickname) {
        if (this.#markers.length === 0) {
            return
        }
        this.map.fitBounds(this.#markers.filter(marker => marker.nickname === nickname).map(marker => marker.getLatLng()));
    }
}

/**
 * The map instance
 */
let map = null;

/**
 * Alias Luxon DateTime
 */
const DateTime = luxon.DateTime;

/**
 * Initializes the map on page load
 */
window.addEventListener('load', () => {
    const publicId = document.getElementById('map').dataset['publicid']
    map = new Maps({publicId: publicId})
    map.display()
    map.getExpiration()
    setInterval(() => {
        map.refreshTTL()
        map.displayLastPosition().then(() => {
            map.displaysParticipants()
        })
    }, 5000,0)

})

/**
 * Zoom the map to display a specific participant
 * @param {string} nickname
 * @returns {void}
 */
zoom = (nickname) => {
    map.fitParticipant(nickname)
}