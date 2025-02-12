function convertToBase64(fileInput, targetId) {
  const file = fileInput.files[0];
  if (file) {
    const reader = new FileReader();
    reader.onload = function (e) {
      document.getElementById(targetId).value = e.target.result;
    };
    reader.readAsDataURL(file);
  }
}

document.addEventListener("DOMContentLoaded", async () => {
  const provinceSelect = document.getElementById("province");
  const citySelect = document.getElementById("city");
  const domisiliInput = document.querySelector("input[name='domisili[]']");

  // Fetch provinces with error handling and logging
  const fetchProvinces = async () => {
    try {
      console.log("Fetching provinces...");
      const response = await fetch("/api/provinces");
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const provinces = await response.json();
      console.log("Provinces data:", provinces);

      // Clear existing options except the first one
      provinceSelect.innerHTML =
        '<option value="" disabled selected>Pilih Provinsi...</option>';

      // Sort provinces alphabetically
      provinces.sort((a, b) => a.name.localeCompare(b.name));

      provinces.forEach((province) => {
        const option = document.createElement("option");
        option.value = province.id;
        option.textContent = province.name;
        provinceSelect.appendChild(option);
      });
    } catch (error) {
      console.error("Error fetching provinces:", error);
      // Show error in the select element
      provinceSelect.innerHTML =
        '<option value="" disabled selected>Error loading provinces</option>';
    }
  };

  // Fetch cities with error handling and logging
  const fetchCities = async (provinceId) => {
    try {
      console.log("Fetching cities for province:", provinceId);
      citySelect.innerHTML =
        '<option value="" disabled selected>Loading...</option>';
      citySelect.disabled = true;

      const response = await fetch(`/api/cities?provinceId=${provinceId}`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const cities = await response.json();
      console.log("Cities data:", cities);

      // Clear and add default option
      citySelect.innerHTML =
        '<option value="" disabled selected>Pilih Kota/Kabupaten...</option>';

      // Sort cities alphabetically
      cities.sort((a, b) => a.name.localeCompare(b.name));

      cities.forEach((city) => {
        const option = document.createElement("option");
        option.value = city.name;
        option.textContent = city.name;
        citySelect.appendChild(option);
      });
      citySelect.disabled = false;
    } catch (error) {
      console.error("Error fetching cities:", error);
      citySelect.innerHTML =
        '<option value="" disabled selected>Error loading cities</option>';
      citySelect.disabled = true;
    }
  };

  // Event listeners
  provinceSelect.addEventListener("change", (e) => {
    const provinceId = e.target.value;
    if (!provinceId) return; // Don't proceed if no province is selected

    const provinceName = e.target.options[e.target.selectedIndex].text;
    fetchCities(provinceId);
    if (domisiliInput) {
      domisiliInput.value = provinceName;
    }
  });

  citySelect.addEventListener("change", (e) => {
    const provinceName =
      provinceSelect.options[provinceSelect.selectedIndex].text;
    const cityName = e.target.value;
    if (provinceName && cityName && domisiliInput) {
      domisiliInput.value = `${provinceName}, ${cityName}`;
    }
  });

  // Initialize provinces
  console.log("Initializing...");
  await fetchProvinces();
});

document.addEventListener("htmx:beforeRequest", function (evt) {
  // Disable button and show loading spinner
  const button = document.getElementById("submit-button");
  const buttonText = document.getElementById("button-text");
  const spinner = document.getElementById("loading-spinner");

  button.disabled = true;
  buttonText.textContent = "Mengirim...";
  spinner.classList.remove("hidden");
});

document.body.addEventListener("htmx:afterRequest", function (evt) {
  // Reset button state
  const button = document.getElementById("submit-button");
  const buttonText = document.getElementById("button-text");
  const spinner = document.getElementById("loading-spinner");

  button.disabled = false;
  buttonText.textContent = "Kirim Lamaran";
  spinner.classList.add("hidden");

  const response = JSON.parse(evt.detail.xhr.response);
  const messageDiv = document.getElementById("response-message");

  if (messageDiv) {
    let customMessage = "";

    if (response.success) {
      customMessage = `<div class="p-4 bg-green-100 text-green-700 rounded">
        🎉 Pendaftaran berhasil! Selamat, akun Anda sudah terdaftar.
      </div>`;
    } else {
      customMessage = `<div class="p-4 bg-red-100 text-red-700 rounded">
        ❌ Pendaftaran gagal: ${response.message}
      </div>`;
    }

    messageDiv.innerHTML = customMessage;
  }

  if (response.success) {
    document.querySelector("form").reset();
  }
});
