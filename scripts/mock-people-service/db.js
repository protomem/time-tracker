/**
 * @typedef {Object} Person
 * @property {string} name - Имя человека
 * @property {string} surname - Фамилия человека
 * @property {string} [patronymic] - Отчество человека (опционально)
 * @property {string} address - Адрес человека
 * @property {string} passport - Серия и номер паспорта человека
 */

/**
 * @type {Array<Person>}
 */
const peopleData = [
  {
    name: "Петр",
    surname: "Петров",
    patronymic: "Петрович",
    address: "г. Санкт-Петербург, пр. Невский, д. 10",
    passport: "4012 345678",
  },

  {
    name: "Сергей",
    surname: "Сидоров",
    patronymic: "Сергеевич",
    address: "г. Екатеринбург, ул. Свердлова, д. 25",
    passport: "6543 210987",
  },

  {
    name: "Анна",
    surname: "Смирнова",
    patronymic: "Андреевна",
    address: "г. Новосибирск, ул. Гоголя, д. 15",
    passport: "9876 543210",
  },

  {
    name: "Олег",
    surname: "Новиков",
    address: "г. Ростов-на-Дону, ул. Советская, д. 12",
    passport: "1234 567890",
  },

  {
    name: "Татьяна",
    surname: "Морозова",
    address: "г. Самара, ул. Ленинградская, д. 5",
    passport: "5678 901234",
  },

  {
    name: "Александр",
    surname: "Волков",
    address: "г. Омск, пр. Мира, д. 18",
    passport: "9012 345678",
  },

  {
    name: "Марина",
    surname: "Соколова",
    address: "г. Челябинск, ул. Труда, д. 7",
    passport: "3456 789012",
  },
];

export default {
  peopleData,
};
